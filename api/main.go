package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"sync"
	"time"

	uri "net/url"

	_ "github.com/mattn/go-sqlite3"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

var (
	once sync.Once
	seed *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
)

const (
	charset      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	shortLength  = 18
	databasePath = "data/tldr.db"
)

type database struct {
	db *sql.DB
}
type Data struct {
	Status  int
	Message string
	Data    Url
}
type Url struct {
	Url   string
	Short string
	Valid int
}

// MakeResponse :: make/build the response data, returns the 'Data' struct.
func MakeResponse(status int, message string, urlData Url) Data {
	data := Data{
		Status:  status,
		Message: message,
		Data:    urlData,
	}
	return data
}

// MakeUrl :: make/build the url data, returns the 'Url' struct with the provided data.
func MakeUrl(url, short string, valid int) Url {
	tmpUrl := Url{
		Url:   url,
		Short: short,
		Valid: valid,
	}
	return tmpUrl
}

// CreateRandomString ::
func CreateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seed.Intn(len(charset))]
	}
	return string(b)
}

// checkDb :: make sure the database is initiated.
func (d database) checkDb() error {
	if d.db == nil {
		return fmt.Errorf("database is not initiated")
	}
	return nil
}

// prepareDatabase :: initialize the database and create a database handle.
//					  This funciton uses the sync.Once method, so the database gets created only once.
func prepareDatabase() (database, error) {
	var d database
	var err error

	prep := func() {
		d.db, err = sql.Open("sqlite3", databasePath)
		if err != nil {
			log.Fatalf("Could not open %s", databasePath)
		}
	}
	once.Do(prep)
	return d, err
}

// GetAllUrls :: as the function name says, retrieve ALL urls and return a map of 'urlRow' structs.
func (d database) GetAllUrls() ([]Url, error) {
	var url []Url

	err := d.checkDb()
	if err != nil {
		return url, err
	}

	query := `SELECT url, short, valid FROM url`
	rows, err := d.db.Query(query)
	if err != nil {
		return url, err
	}

	// Loop over all the returned data, prepare the struct, fill it with data and append it to the map.
	for rows.Next() {
		var tmp Url
		err = rows.Scan(&tmp.Url, &tmp.Short, &tmp.Valid)
		if err != nil {
			log.Printf("ERROR: %s", err.Error())
			return url, err
		}
		url = append(url, tmp)
	}

	return url, err
}

// GetUrlFromShort :: this function resolves the `short` and returns the 'urlRow' struct filled with
//					  the data from the database.
func (d database) GetUrlFromShort(urlShort string) (bool, Url, error) {
	var url Url
	query := `SELECT url, short, valid FROM url WHERE short=$1`

	err := d.checkDb()
	if err != nil {
		return false, url, err
	}

	// Query for a single row.
	row := d.db.QueryRow(query, urlShort)
	switch err := row.Scan(&url.Url, &url.Short, &url.Valid); err {
	case sql.ErrNoRows:
		return false, url, nil
	case nil:
		return true, url, nil
	default:
		log.Printf("ERROR: %s", err.Error())
		return false, url, err
	}
}

// InsertNewUrl :: insert a new url into the database.
func (d database) InsertNewUrl(url Url) error {
	query := `INSERT INTO url (url, short, valid) VALUES (?, ?, ?)`

	err := d.checkDb()
	if err != nil {
		return err
	}

	// Prepare the sql statement, this prevents sql injections.
	sqlStmt, err := d.db.Prepare(query)
	if err != nil {
		return err
	}

	// Execute the prepared statement.
	_, err = sqlStmt.Exec(url.Url, url.Short, url.Valid)
	if err != nil {
		return err
	}

	return nil
}

// PrepareNewUrl :: create a new short and make sure that it doesn't already exists.
func (d database) PrepareNewUrl(url string) (Url, error) {
	var short string
	var resp Url
	ok := false

	// Generate a new short, make sure the short isn't already in use.
	for !ok {
		tmpShort := CreateRandomString(shortLength)
		valid, _, err := d.GetUrlFromShort(tmpShort)
		if err != nil {
			return resp, err
		} else if !valid {
			short = tmpShort
			ok = true
		}
	}
	resp = MakeUrl(url, short, 1)

	return resp, nil
}

// IsValidHttpUrl :: make sure the provided url is a valid http address.
func IsValidHttpUrl(url string) (bool, error) {
	match, err := regexp.MatchString(`^http?://`, url)
	if err != nil {
		return false, err
	}
	if !match {
		return false, nil
	}
	return true, nil
}

// IsValidHttpsUrl :: make sure the provided url is a valid https address.
func IsValidHttpsUrl(url string) (bool, error) {
	match, err := regexp.MatchString(`^https?://`, url)
	if err != nil {
		return false, err
	}
	if !match {
		return false, nil
	}
	return true, nil
}

// IsValid :: returns true if url from provided struct is valid, else returns false.
func IsValid(url Url) bool {
	return url.Valid == 1
}

func main() {
	db, err := prepareDatabase()
	if err != nil {
		panic(err)
	}
	app := fiber.New()

	// Register middleware, precerve the requestID and also create a backend logger with a specific format.
	app.Use(favicon.New())
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format:     "${pid} - ${locals:requestid} :: [${status}] - ${method} - ${path}\n",
		TimeFormat: "Jan-02-2006",
		TimeZone:   "Europe/Vienna",
	}))

	// Base /api/ route, returns ALL the available/registered routes/urls.
	app.Get("/api/", func(c *fiber.Ctx) error {
		urlMap, err := db.GetAllUrls()
		if err != nil {
			log.Printf("ERROR: %s", err.Error())
			data := MakeResponse(500, err.Error(), Url{})
			return c.JSON(data)
		}

		// Filter the db response and create a payload to send back.
		var data []Data
		for i := 0; i < len(urlMap); i++ {
			var resp Data

			url := MakeUrl(urlMap[i].Url, urlMap[i].Short, urlMap[i].Valid)
			if IsValid(urlMap[i]) {
				resp = MakeResponse(200, "Ok", url)
			} else {
				resp = MakeResponse(422, "URL is not valid", url)
			}
			data = append(data, resp)
		}

		return c.JSON(data)
	})

	// Create new shorts, send a payload containing the url you want to be shortened.
	// Post body example:
	// {
	//		"url": "example-domain.com"
	// }
	app.Post("/api/", func(c *fiber.Ctx) error {
		var err error
		var data Data
		type urlPost struct {
			Url string `json:"url"`
		}
		url := new(urlPost)

		// Parse the retrieved body content to the newly created struct.
		if err = c.BodyParser(url); err != nil {
			log.Printf("ERROR: %s", err.Error())
			data = MakeResponse(500, err.Error(), Url{})
			return c.JSON(data)
		}

		// Make sure that the provided url is an actuall url that can get redirected to (http|https).
		https, err := IsValidHttpsUrl(url.Url)
		if err != nil {
			log.Printf("ERROR: %s", err.Error())
		}
		http, err := IsValidHttpUrl(url.Url)
		if err != nil {
			log.Printf("ERROR: %s", err.Error())
		}
		if !https && !http {
			log.Printf("WARN: URL (%s) does not have a http* prefix, adding https:// to it", url.Url)
			url.Url = "https://" + url.Url
		}
		// Check if it's parseable.
		_, err = uri.ParseRequestURI(url.Url)
		if err != nil {
			log.Printf("ERROR: %s", err.Error())
			data = MakeResponse(500, err.Error(), Url{})
			return c.JSON(data)
		}

		// Prepare the new url for insertion.
		prepUrl, err := db.PrepareNewUrl(url.Url)
		if err != nil {
			log.Printf("ERROR: %s", err.Error())
			data = MakeResponse(500, err.Error(), Url{})
			return c.JSON(data)
		}

		// Insert the new url.
		err = db.InsertNewUrl(prepUrl)
		if err != nil {
			log.Printf("ERROR: %s", err.Error())
			data = MakeResponse(500, err.Error(), Url{})
			return c.JSON(data)
		}

		// Send the 200 OK with the newly created url.
		data = MakeResponse(200, "Ok", prepUrl)
		return c.JSON(data)
	})

	// This route get's invoked with a paramaeter (the short to unvail).
	// It requests the given parameter (short url) and returns the redirect url.
	app.Get("/api/*", func(c *fiber.Ctx) error {
		var url Url
		var param string
		var data Data

		param = c.Params("*")
		found, url, err := db.GetUrlFromShort(param)
		if err != nil {
			log.Printf("ERROR: %s", err.Error())
			data := MakeResponse(500, err.Error(), Url{})
			return c.JSON(data)
		} else if !found {
			msg := fmt.Sprintf("No URL found for short '%s'.", param)
			data := MakeResponse(404, msg, Url{})
			return c.JSON(data)
		}
		// Make sure the URL is valid..
		if !IsValid(url) {
			data = MakeResponse(422, "URL is not valid", Url{})
			return c.JSON(data)
		}
		data = MakeResponse(200, "Ok", url)
		return c.JSON(data)
	})

	log.Fatal(app.Listen(":3000"))
}
