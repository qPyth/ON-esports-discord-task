package pkg

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/viper"
	"log/slog"
	"sync"
	"time"
)

type AwsS3 struct {
	region     string
	bucketName string
}

func NewAwsS3(cfg AWSConfig) *AwsS3 {
	return &AwsS3{region: cfg.Region, bucketName: cfg.BucketName}
}

func (s AwsS3) Save(data []byte, name string) error {
	op := "AWSS3.Save"
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(s.region),
		Credentials: credentials.NewStaticCredentials(viper.GetString("aws_access_key"),
			viper.GetString("aws_secret_access_key"), "")},
	)
	if err != nil {
		return fmt.Errorf("%s session create error: %w", op, err)
	}

	s3Client := s3.New(sess)

	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(name),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("%s put object error: %w", op, err)
	}

	return nil
}

func (s AwsS3) GenerateLinks(folder string) ([]string, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(s.region),
		Credentials: credentials.NewStaticCredentials(viper.GetString("aws_access_key"), viper.GetString("aws_secret_access_key"), ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	s3Client := s3.New(sess)
	var fileKeys []string

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(folder),
	}

	err = s3Client.ListObjectsV2Pages(input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, object := range page.Contents {
			fileKeys = append(fileKeys, *object.Key)
		}
		return !lastPage
	})
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var urls []string

	for _, key := range fileKeys {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			req, _ := s3Client.GetObjectRequest(&s3.GetObjectInput{
				Bucket: aws.String(s.bucketName),
				Key:    aws.String(path),
			})
			urlStr, err := req.Presign(15 * time.Minute)
			if err != nil {
				slog.Error("presign error", "err", err.Error())
				return
			}
			mu.Lock()
			urls = append(urls, urlStr)
			mu.Unlock()
		}(key)
	}

	wg.Wait()
	return urls, nil
}
