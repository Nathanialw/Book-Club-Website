package main

import (
	"database/sql"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/nfnt/resize"
)

func textNotEmpty(title string) bool {
	if title == "" {
		fmt.Printf("%s is empty\n", title)
		return false
	}
	return true
}

func validYear(year string, era string) bool {
	if year == "" || era == "" {
		fmt.Println("year is empty")
		return false
	}
	for _, c := range year {
		if c < '0' || c > '9' {
			fmt.Println("year is not a number")
			return false
		}
	}
	if era != "BC" && era != "AD" {
		fmt.Println("era is not BC or AD")
		return false
	}
	if len(year) > 4 {
		fmt.Println("year is more than 4 characters")
		return false
	}
	if year == "0" {
		fmt.Println("year is no year 0 BC or 0 AD")
		return false
	}
	if era == "AD" && year > "2024" {
		fmt.Println("the max year is 2024 AD")
		return false
	}
	return true
}

func verifyImage(handler *multipart.FileHeader) bool {
	// Check the file type
	fileType := handler.Header.Get("Content-Type")
	switch fileType {
	case "image/jpeg", "image/jpg", "image/png":
		break
	default:
		fmt.Println("File is not an image")
		return false
	}
	if handler.Size > 4*1024*1024 {
		fmt.Printf("File is too large (max 4MB) %d\n", handler.Size)
		return false
	}
	//check if the file is an image
	if !strings.Contains(handler.Header.Get("Content-Type"), "image") {
		fmt.Printf("file is not an image: %s\n", handler.Header.Get("Content-Type"))
		return false
	}
	//check if the file already exists
	if _, err := os.Stat("/assets/images/book_covers/" + handler.Filename); err == nil {
		//maybe append a number to the filename?
		fmt.Printf("file already exists: %s\n", handler.Filename)
		return false
	}

	return true
}

func VerifyAndInsertBook(w http.ResponseWriter, r *http.Request, db *sql.DB) bool {
	//verify book data is formatted correctly
	title := r.FormValue("title")
	subtitle := r.FormValue("subtitle")
	author := r.FormValue("author")
	publish_year := r.FormValue("publish_year")
	publish_era := r.FormValue("publish_era")
	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		fmt.Printf("error parsing the form: %s\n", err)
		return false
	}
	// Retrieve the file from form data
	file, handler, err := r.FormFile("image") // "image" is the name of the file input field
	var filename string
	if err != nil {
		fmt.Printf("error retrieving the file, setting empty: %s\n", err)
		filename = ""
	} else {
		filename = handler.Filename
	}
	synopsis := r.FormValue("synopsis")

	//link_amazon := r.FormValue("link_amazon")
	//link_indigo := r.FormValue("link_indigo")
	//link_pdf := r.FormValue("link_pdf")
	//link_epub := r.FormValue("link_epub")
	//link_handmade := r.FormValue("link_handmade")
	//link_text := r.FormValue("link_text")

	//verify title
	if !textNotEmpty(title) {
		fmt.Printf("0: %s\n", title)
		return false
	}
	//verify author
	if !textNotEmpty(author) {
		fmt.Printf("2: %s\n", title)
		return false
	}
	//verify publish_year
	if !textNotEmpty(synopsis) {
		if !validYear(publish_year, publish_era) {
			fmt.Printf("3: %s\n", title)
			return false
		}
	}
	//verify synopsis
	if !textNotEmpty(synopsis) {
		fmt.Printf("4: %s\n", title)
		return false
	}

	//check book is already in the database
	fmt.Printf("5: %s\n", title)
	if db == contentDB {
		rows, err := db.Query("select title from books where title = ? and subtitle = ? and author = ?", title, subtitle, author)
		if err != nil {
			fmt.Printf("error checking if book exists: %s\n", err)
			return false
		}
		for rows.Next() {
			fmt.Printf("book already exists: %s\n", title)
			return false
		}
	}

	//verify image
	var imagePath string
	var imagePath100 string
	var imagePath400 string
	if !textNotEmpty(filename) {
		fmt.Print("image field empty\n")
		if db == contentDB {
			fmt.Print("failed to add book to content\n")
			return false
		} else {

			//need to get the UID of the book somehow

			rows, err := db.Query("select title from imagePath, imagePath100, imagePath400 where uid = ?")
			if err != nil {
				fmt.Printf("error image does not exist in contentDB exists: %s\n", err)
				return false
			}
			for rows.Next() {
				rows.Scan(&imagePath, &imagePath100, &imagePath400)
				fmt.Print("add book image to submit book\n")
				//imagePath = imagePath
				//imagePath100 = imagePath100
				//imagePath400 = imagePath400
			}
		}
	} else {
		// if either fb handler is not empty
		if !verifyImage(handler) {
			return false
		}
		// Create the file in the file system
		systemPath := "../../public/assets/images/book_covers/" + handler.Filename
		dst, err := os.Create(systemPath)
		if err != nil {
			fmt.Printf("error creating the file: %s\n", err)
			return false
		}
		defer dst.Close()
		// Copy the uploaded file to the filesystem at the specified destination
		_, err = io.Copy(dst, file)
		if err != nil {
			fmt.Printf("error copying the file: %s\n", err)
			return false
		}

		OSFile, _ := os.Open(systemPath)
		defer file.Close()

		//resize the image
		img, _, err := image.Decode(OSFile)
		if err != nil {
			fmt.Printf("error decoding the image: %s, %s\n", err, systemPath)
			return false
		}

		extension := filepath.Ext(systemPath)
		switch extension {
		case ".jpg", ".jpeg":
			fmt.Printf("jpeg\n")
			OSFile.Seek(0, 0) // Reset the reader to the start of the file
			img, err = jpeg.Decode(OSFile)
			//resize the image to 400x400
			m := resize.Resize(400, 0, img, resize.Lanczos3)
			systemPath = "../../public/assets/images/book_covers/400_" + handler.Filename
			out, _ := os.Create(systemPath)
			defer out.Close()
			// Write the new image to the new file
			jpeg.Encode(out, m, nil)

			m = resize.Resize(100, 0, img, resize.Lanczos3)
			systemPath = "../../public/assets/images/book_covers/100_" + handler.Filename
			out, _ = os.Create(systemPath)
			jpeg.Encode(out, m, nil)
		case ".png":
			fmt.Printf("png\n")
			OSFile.Seek(0, 0) // Reset the reader to the start of the file
			img, err = png.Decode(OSFile)
			//resize the image to 400x400
			m := resize.Resize(400, 0, img, resize.Lanczos3)
			systemPath = "../../public/assets/images/book_covers/400_" + handler.Filename
			out, _ := os.Create(systemPath)
			defer out.Close()
			//write the new image to the new file
			png.Encode(out, m)

			m = resize.Resize(100, 0, img, resize.Lanczos3)
			systemPath = "../../public/assets/images/book_covers/100_" + handler.Filename
			out, _ = os.Create(systemPath)
			png.Encode(out, m)
		case ".gif":
			fmt.Printf("unsupported image format: %s\n", extension)
			return false
			//OSFile.Seek(0, 0) // Reset the reader to the start of the file
			//img, err = gif.Decode(OSFile)
			////resize the image to 400x400
			//m := resize.Resize(400, 0, img, resize.Lanczos3)
			//systemPath = "../../public/assets/images/book_covers/400_" + handler.Filename
			//out, _ := os.Create(systemPath)
			//defer out.Close()
			//// Write the new image to the new file
			//gif.Encode(out, m, nil)
			//
			//m = resize.Resize(100, 0, img, resize.Lanczos3)
			//systemPath = "../../public/assets/images/book_covers/100_" + handler.Filename
			//out, _ = os.Create(systemPath)
			//gif.Encode(out, m, nil)
		default:
			fmt.Printf("unsupported image format: %s\n", extension)
			return false
		}

		//add book to database
		imagePath = "/assets/images/book_covers/" + handler.Filename
		imagePath100 = "/assets/images/book_covers/100_" + handler.Filename
		imagePath400 = "/assets/images/book_covers/400_" + handler.Filename
	}

	_, err = db.Exec("insert into books (title, subtitle, author, publish_year, publish_era, image, image_100, image_400, synopsis) values (?, ?, ?, ?, ?, ?, ?, ?, ?)", title, subtitle, author, publish_year, publish_era, imagePath, imagePath100, imagePath400, synopsis)
	if err != nil {
		fmt.Printf("error adding book: %s\n", err)
		return false
	}

	fmt.Printf("successfully added book: %s\n", title)
	return true
}
