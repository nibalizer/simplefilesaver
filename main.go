package main

// Utility to automatically backup factorio save files
// Intends to be run as a sidecar with the factorio server

// much of this from the fsnotify examples and minio examples

import (
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io/ioutil"
	"log"
	"os"
	"sort"
)

func backup(mc *minio.Client, ctx context.Context, bucketName string, dir string) error {

	files := sortedFileList(dir)
	latestSavePath := dir + "/" + files[0].Name()
	objectName := fmt.Sprintf("_autosaveXX-%d.zip", files[0].ModTime().Unix())
	fmt.Println(bucketName, objectName, latestSavePath)

	n, err := mc.FPutObject(ctx, bucketName, objectName, latestSavePath, minio.PutObjectOptions{ContentType: "application/zip"})
	if err != nil {
		return err
	}
	log.Printf("Successfully uploaded %s of size %d\n", objectName, n)
	return nil
}

func sortedFileList(dir string) []os.FileInfo {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	sort.Slice(files, func(a, b int) bool {
		return files[a].ModTime().After(files[b].ModTime())
	})
	fmt.Println(files)
	/*
		// in case I ever doubt the above code
		for _, file := range files {
			fmt.Println(file.ModTime(), file.Name())
		}
	*/
	return files
}

func main() {
	fmt.Println("-=Factorio Save Backuper Sidecar=-")
	ctx := context.Background()
	// S3_ENDPOINT="localhost:9000"
	// S3_ENDPOINT="minio.minio:9000"
	endpiont := os.Getenv("S3_ENDPOINT")
	// ACCESS_KEY_ID="xxx"
	accessKeyID := os.Getenv("ACCESS_KEY_ID")
	// SECRET_ACCESS_KEY="xxx"
	secretAccessKey := os.Getenv("SECRET_ACCESS_KEY")
	// SAVES_DIRECTORY="/factorio/saves"
	savesDirectory := os.Getenv("SAVES_DIRECTORY")
	// FSID="abcdabcd"
	// 'factorio server id'
	FSID := os.Getenv("FSID")

	fmt.Println("Watching save files in:", savesDirectory)

	// use this in the future
	useSSL := false

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})

	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("%#v\n", minioClient) // minioClient is now set up

	// Make a new bucket called mymusic.
	bucketName := "factorio-saves-" + FSID
	location := "us-east-1"

	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		// Check to see if we already own this bucket
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", bucketName)
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Printf("Successfully created %s\n", bucketName)
	}

	//inotify yeet
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					// Fire backups on the "remove" event

					// "remove" catches the final step in the factorio backup sequence
					// after autosave3.new.zip replaces autosave3.zip and then
					// autosave3.bak.zip is removed. Thus we know the file is completely
					// written and the newest file in the directory is the latest save
					// file
					log.Println("removed a file:", event.Name)
					log.Println("Running backup code")
					err = backup(minioClient, ctx, bucketName, savesDirectory)
					if err != nil {
						log.Fatal(err)
					}
				}
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(savesDirectory)
	if err != nil {
		log.Fatal(err)
	}
	<-done

}
