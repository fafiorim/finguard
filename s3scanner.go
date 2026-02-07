package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	amaasclient "github.com/trendmicro/tm-v1-fs-golang-sdk"
)

var s3Logger *log.Logger

func initS3Logger() {
	logFile, err := os.OpenFile("/var/log/s3-scanner.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Failed to open S3 log file: %v", err)
		s3Logger = log.New(os.Stdout, "[S3] ", log.LstdFlags)
	} else {
		s3Logger = log.New(io.MultiWriter(logFile, os.Stdout), "[S3] ", log.LstdFlags)
	}
	s3Logger.Println("=== S3 Scanner initialized ===")
}

// S3ClientReader implements AmaasClientReader for S3 objects
type S3ClientReader struct {
	client *s3.Client
	bucket string
	key    string
	size   int64
}

func NewS3ClientReader(ctx context.Context, awsAccessKey, awsSecretKey, bucketRegion, bucket, key string) (*S3ClientReader, error) {
	s3Logger.Printf("Creating S3 reader for s3://%s/%s in region %s", bucket, key, bucketRegion)

	var cfg aws.Config
	var err error

	// Load config with credentials if provided
	if awsAccessKey != "" && awsSecretKey != "" {
		s3Logger.Println("Using provided AWS credentials")
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(bucketRegion),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(awsAccessKey, awsSecretKey, "")),
		)
	} else {
		s3Logger.Println("Using default AWS credentials from environment")
		// Use default credentials from environment
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(bucketRegion))
	}

	if err != nil {
		s3Logger.Printf("Failed to load AWS config: %v", err)
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	s3Logger.Println("AWS S3 client created successfully")

	// Get object attributes to determine size
	s3Logger.Printf("Getting object attributes for %s", key)
	attr, err := client.GetObjectAttributes(ctx, &s3.GetObjectAttributesInput{
		Bucket: &bucket,
		Key:    &key,
		ObjectAttributes: []types.ObjectAttributes{
			types.ObjectAttributesObjectSize,
		},
	})
	if err != nil {
		s3Logger.Printf("Failed to get object attributes: %v", err)
		return nil, err
	}

	if attr.ObjectSize == nil {
		s3Logger.Println("Object size is nil")
		return nil, fmt.Errorf("unable to get object size from S3")
	}

	s3Logger.Printf("Object size: %d bytes", *attr.ObjectSize)
	return &S3ClientReader{
		client: client,
		bucket: bucket,
		key:    key,
		size:   *attr.ObjectSize,
	}, nil
}

// Identifier returns the S3 object identifier
func (r *S3ClientReader) Identifier() string {
	return fmt.Sprintf("s3://%s/%s", r.bucket, r.key)
}

// DataSize returns the size of the S3 object
func (r *S3ClientReader) DataSize() (int64, error) {
	return r.size, nil
}

// ReadBytes reads bytes from the S3 object at the specified offset
func (r *S3ClientReader) ReadBytes(offset int64, length int32) ([]byte, error) {
	rng := fmt.Sprintf("bytes=%d-%d", offset, offset+int64(length)-1)

	output, err := r.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &r.bucket,
		Key:    &r.key,
		Range:  &rng,
	})
	if err != nil {
		return nil, err
	}
	defer output.Body.Close()

	bytes, err := io.ReadAll(output.Body)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error reading S3 object body: %v", err)
	}

	return bytes, err
}

// getBucketRegion detects the region of an S3 bucket
func getBucketRegion(ctx context.Context, cfg aws.Config, bucket string) (string, error) {
	client := s3.NewFromConfig(cfg)
	resp, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return "", err
	}

	// GetBucketLocation returns empty string for us-east-1
	if resp.LocationConstraint == "" {
		return "us-east-1", nil
	}
	return string(resp.LocationConstraint), nil
}

// HTTP handler for listing S3 buckets
func handleListBuckets(scannerClient *amaasclient.AmaasClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		s3Logger.Printf("--- LIST BUCKETS REQUEST at %s ---", time.Now().Format(time.RFC3339))

		var req struct {
			AwsAccessKey string `json:"awsAccessKey"`
			AwsSecretKey string `json:"awsSecretKey"`
			Region       string `json:"region"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		var cfg aws.Config
		var err error

		if req.AwsAccessKey != "" && req.AwsSecretKey != "" {
			cfg, err = config.LoadDefaultConfig(ctx,
				config.WithRegion(req.Region),
				config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(req.AwsAccessKey, req.AwsSecretKey, "")),
			)
		} else {
			cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(req.Region))
		}

		if err != nil {
			s3Logger.Printf("ERROR: Failed to load AWS config: %v", err)
			http.Error(w, fmt.Sprintf("Failed to load AWS config: %v", err), http.StatusInternalServerError)
			return
		}

		client := s3.NewFromConfig(cfg)
		s3Logger.Println("Listing S3 buckets...")
		result, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
		if err != nil {
			s3Logger.Printf("ERROR: Failed to list buckets: %v", err)
			http.Error(w, fmt.Sprintf("Failed to list buckets: %v", err), http.StatusInternalServerError)
			return
		}
		s3Logger.Printf("Found %d buckets", len(result.Buckets))

		buckets := make([]map[string]interface{}, 0)
		for _, bucket := range result.Buckets {
			s3Logger.Printf("  - Bucket: %s (created: %s)", *bucket.Name, bucket.CreationDate)
			buckets = append(buckets, map[string]interface{}{
				"name":         *bucket.Name,
				"creationDate": bucket.CreationDate,
			})
		}
		s3Logger.Printf("Successfully listed %d buckets", len(buckets))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"buckets": buckets,
		})
	}
}

// HTTP handler for listing S3 objects in a bucket
func handleListObjects(scannerClient *amaasclient.AmaasClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		s3Logger.Printf("--- LIST OBJECTS REQUEST at %s ---", time.Now().Format(time.RFC3339))

		var req struct {
			AwsAccessKey string `json:"awsAccessKey"`
			AwsSecretKey string `json:"awsSecretKey"`
			Region       string `json:"region"`
			Bucket       string `json:"bucket"`
			Prefix       string `json:"prefix"`
			Recursive    bool   `json:"recursive"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		var cfg aws.Config
		var err error

		if req.AwsAccessKey != "" && req.AwsSecretKey != "" {
			cfg, err = config.LoadDefaultConfig(ctx,
				config.WithRegion(req.Region),
				config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(req.AwsAccessKey, req.AwsSecretKey, "")),
			)
		} else {
			cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(req.Region))
		}

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load AWS config: %v", err), http.StatusInternalServerError)
			return
		}

		client := s3.NewFromConfig(cfg)

		// Try to get bucket region first
		bucketRegion, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
			Bucket: &req.Bucket,
		})
		if err != nil {
			log.Printf("Warning: Could not get bucket region for %s: %v", req.Bucket, err)
		} else if bucketRegion.LocationConstraint != "" {
			// Recreate client with correct region
			log.Printf("Bucket %s is in region: %s", req.Bucket, bucketRegion.LocationConstraint)
			if req.AwsAccessKey != "" && req.AwsSecretKey != "" {
				cfg, err = config.LoadDefaultConfig(ctx,
					config.WithRegion(string(bucketRegion.LocationConstraint)),
					config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(req.AwsAccessKey, req.AwsSecretKey, "")),
				)
			} else {
				cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(string(bucketRegion.LocationConstraint)))
			}
			if err == nil {
				client = s3.NewFromConfig(cfg)
			}
		}

		var prefix *string
		if req.Prefix != "" {
			prefix = &req.Prefix
		}

		log.Printf("Listing objects in bucket %s with prefix '%s' (recursive: %v)", req.Bucket, req.Prefix, req.Recursive)

		objects := make([]map[string]interface{}, 0)
		var continuationToken *string

		// Paginate through all results
		for {
			result, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
				Bucket:            &req.Bucket,
				Prefix:            prefix,
				ContinuationToken: continuationToken,
			})
			if err != nil {
				log.Printf("Failed to list objects in %s: %v", req.Bucket, err)
				http.Error(w, fmt.Sprintf("Failed to list objects: %v", err), http.StatusInternalServerError)
				return
			}

			for _, obj := range result.Contents {
				// If not recursive, skip objects that are in subdirectories
				if !req.Recursive && req.Prefix != "" {
					relativePath := (*obj.Key)[len(req.Prefix):]
					if strings.Contains(relativePath, "/") {
						continue
					}
				} else if !req.Recursive && req.Prefix == "" {
					if strings.Contains(*obj.Key, "/") {
						continue
					}
				}

				s3Logger.Printf("  - Object: %s (size: %d bytes)", *obj.Key, obj.Size)
				objects = append(objects, map[string]interface{}{
					"key":          *obj.Key,
					"size":         obj.Size,
					"lastModified": obj.LastModified,
				})
			}

			// Check if there are more results
			if !*result.IsTruncated {
				break
			}
			continuationToken = result.NextContinuationToken
		}

		s3Logger.Printf("Successfully listed %d objects from s3://%s/%s", len(objects), req.Bucket, req.Prefix)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"bucket":  req.Bucket,
			"objects": objects,
		})
	}
}

// HTTP handler for scanning S3 objects
func handleScanS3Object(scannerClient *amaasclient.AmaasClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		s3Logger.Printf("=== SCAN REQUEST at %s ===", time.Now().Format(time.RFC3339))

		var req struct {
			AwsAccessKey string   `json:"awsAccessKey"`
			AwsSecretKey string   `json:"awsSecretKey"`
			Region       string   `json:"region"`
			Bucket       string   `json:"bucket"`
			Key          string   `json:"key"`
			Tags         []string `json:"tags"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s3Logger.Printf("Invalid request body: %v", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		s3Logger.Printf("Scan target: s3://%s/%s", req.Bucket, req.Key)
		s3Logger.Printf("Region: %s, Tags: %v", req.Region, req.Tags)

		ctx := context.Background()

		// Create S3 reader
		s3Logger.Println("Creating S3 reader for scan...")
		reader, err := NewS3ClientReader(ctx, req.AwsAccessKey, req.AwsSecretKey, req.Region, req.Bucket, req.Key)
		if err != nil {
			s3Logger.Printf("ERROR: Failed to create S3 reader: %v", err)
			http.Error(w, fmt.Sprintf("Failed to create S3 reader: %v", err), http.StatusInternalServerError)
			return
		}
		s3Logger.Println("S3 reader created successfully")

		// Scan the S3 object using the scanner client
		tags := req.Tags
		if tags == nil {
			tags = []string{"source:s3"}
		} else {
			tags = append(tags, "source:s3")
		}

		log.Printf("=== Starting S3 Scan ===")
		log.Printf("Object: s3://%s/%s", req.Bucket, req.Key)
		log.Printf("Region: %s", req.Region)
		log.Printf("Size: %d bytes", reader.size)

		scanResult, err := scannerClient.ScanReader(reader, tags)
		if err != nil {
			log.Printf("❌ Scan FAILED for s3://%s/%s: %v", req.Bucket, req.Key, err)
			http.Error(w, fmt.Sprintf("Scan failed: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("✓ Scan COMPLETED successfully for s3://%s/%s", req.Bucket, req.Key)
		log.Printf("Result preview: %s", scanResult[:min(len(scanResult), 200)])

		// Parse scan result to extract key information
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(scanResult), &result); err != nil {
			s3Logger.Printf("WARNING: Failed to parse scan result: %v", err)
		} else {
			if scanResultCode, ok := result["scanResult"].(float64); ok {
				if scanResultCode == 0 {
					s3Logger.Printf("Scan result: CLEAN (no threats detected)")
				} else {
					s3Logger.Printf("Scan result: THREAT DETECTED (code: %.0f)", scanResultCode)
					if foundMalwares, ok := result["foundMalwares"].([]interface{}); ok {
						s3Logger.Printf("  Found %d malware(s):", len(foundMalwares))
						for _, malware := range foundMalwares {
							if m, ok := malware.(map[string]interface{}); ok {
								s3Logger.Printf("    - %s in %s", m["malwareName"], m["fileName"])
							}
						}
					}
				}
			}
			if scanId, ok := result["scanId"].(string); ok {
				s3Logger.Printf("Scan ID: %s", scanId)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"scanResult": scanResult,
			"bucket":     req.Bucket,
			"key":        req.Key,
			"region":     req.Region,
		})
	}
}
