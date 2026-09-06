package main

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func GeneratePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignedClient := s3.NewPresignClient(s3Client)

	presignedHttpReq, err := presignedClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{}, s3.WithPresignExpires(expireTime))

	if err != nil {
		return "", err
	}

	return presignedHttpReq.URL, nil
}
