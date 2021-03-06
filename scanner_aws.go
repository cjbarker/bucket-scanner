package bucketscanner

import (
	"encoding/xml"
	"errors"
	"net/http"
	"strings"
	"time"
)

// Cloud Provider Bucket Constant
const (
	awsName = "Amazon Simple Storage Service (S3)"
	awsURI  = "https://" + bucketName + ".s3.amazonaws.com"
)

// AwsScanner is struct for cloud scanner of Amazon Web Services
type AwsScanner struct {
	result ListBucketResult
}

// ListBucketResult is the analyzed results read from a given AWS bucket
type ListBucketResult struct {
	XMLName      xml.Name `xml:"ListBucketResult"`
	Name         string
	Prefix       string
	Marker       string
	MaxKeys      int
	IsTruncated  bool
	ContentsList []Contents `xml:"Contents"`
}

// Contents are the XML contents of the bucket (one per file)
type Contents struct {
	XMLName      xml.Name `xml:"Contents"`
	Key          string
	LastModified string
	Etag         string `xml:"ETag"`
	Size         int
	StorageClass string
}

// Read establishes HTTP connection and reads the contents from the bucket
func (a AwsScanner) Read(name string) (bucket *Bucket, err error) {
	if strings.Trim(name, " ") == "" {
		return nil, errors.New("Blank strings not accepted for bucket name")
	}

	url := strings.Replace(awsURI, bucketName, name, 1)

	bucket = &Bucket{
		Provider: awsName,
		Name:     name,
		URI:      url,
		State:    Unknown,
		Scanned:  time.Now(),
	}

	var sleepMs int

	// Parse State
	for bucket.State == Unknown {
		// Head check before deeper analysis
		resp, err := http.Head(url)
		if err != nil {
			return nil, err
		}

		switch resp.StatusCode {
		case 200:
			bucket.State = Public
		case 403:
			bucket.State = Private
		case 404:
			bucket.State = Invalid
		case 503:
			sleepMs += 500
			time.Sleep(time.Duration(sleepMs) * time.Millisecond)
			if sleepMs >= 10000 {
				bucket.State = RateLimited
			}
		default:
			bucket.State = Unknown
		}
	}

	// Retrieve available HTTP payload
	if bucket.State == Public {
		contents, err := getHTTPBucket(url)
		if err != nil {
			return nil, err
		}

		//fmt.Printf("Resp. Bucket Contents: %s\n", *contents)

		err = xml.Unmarshal([]byte(*contents), &a.result)
		if err != nil {
			return nil, err
		}

		/*
			fmt.Printf("Result: %s\n", a.result.Name)
			fmt.Printf("Result: %d\n", a.result.MaxKeys)
			fmt.Printf("Result: %t\n", a.result.IsTruncated)
		*/

		for _, element := range a.result.ContentsList {
			bucket.NoFiles++
			bucket.TotalSize += int64(element.Size)

			var isDir = false
			if strings.HasSuffix(element.Key, "/") {
				isDir = true
			}

			bucketFile := file{
				Name:  element.Key,
				Size:  int64(element.Size),
				IsDir: isDir,
			}

			bucket.Files = append(bucket.Files, bucketFile)

			/*
				fmt.Printf("\nResult: %s\n", element.Key)
				fmt.Printf("Result: %s\n", element.LastModified)
				fmt.Printf("Result: %s\n", element.Etag)
				fmt.Printf("Result: %d\n", element.Size)
				fmt.Printf("Result: %s\n", element.StorageClass)
			*/

		}
	}

	return bucket, nil
}

// Write attempts to write a temporary file to a given bucket within AWS
func (a AwsScanner) Write(name string) (isWritable bool, err error) {
	if strings.Trim(name, " ") == "" {
		return false, errors.New("Blank strings not accepted for bucket name")
	}

	//url := strings.Replace(azureURI, bucketName, name, 1)

	// TODO implement

	return false, errors.New("AWSWriter is currently not supported")
}

// GetProviderName returns the given Cloud Provider's name for the scanner
func (a AwsScanner) GetProviderName() (cloudProviderName string) {
	return awsName
}
