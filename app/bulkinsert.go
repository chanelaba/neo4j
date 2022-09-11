package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/joho/godotenv"
)

type Neo4jConfiguration struct {
	Url      string
	Username string
	Password string
	Database string
}

func (nc *Neo4jConfiguration) newDriver() (neo4j.Driver, error) {
	return neo4j.NewDriver(nc.Url, neo4j.BasicAuth(nc.Username, nc.Password, ""))
}

func bulkInsertCEO(driver neo4j.Driver, database string) {

	session := driver.NewSession(neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: database,
	})
	defer unsafeClose(session)

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		_, err := tx.Run(
			`MATCH (ceo:Ceo)
			DETACH DELETE ceo`,
			nil)
		if err != nil {
			return nil, err
		}
		result, err2 := tx.Run(
			`LOAD CSV WITH HEADERS FROM $file_name AS row
			CREATE (:Ceo {
				previous: row.previous,
				name: row.name,
				company: row.company
			})`,
			map[string]interface{}{"file_name": "file:///" + lookupEnvOrGetDefault("NEO4J_IMPORT_CSV_CEO", "")})
		if err2 != nil {
			return nil, err2
		}
		return result, nil
	})
	if err != nil {
		log.Println("error bulk inserting:", err)
		return
	}
}

func main() {
	log.Println("Bulk Insert Start.")

	err := godotenv.Load(fmt.Sprintf("./env/%s.env", os.Getenv("GO_ENV")))
    if err != nil {
        log.Println("error load env. please check env file path")
		log.Println(os.Getenv("GO_ENV"))
    }
	configuration := parseConfiguration()
	driver, err := configuration.newDriver()
	if err != nil {
		log.Fatal(err)
	}
	defer unsafeClose(driver)

	// bulk insert ceo
	bulkInsertCEO(driver, configuration.Database)

	log.Println("Bulk Insert End.")
	
}

func parseConfiguration() *Neo4jConfiguration {
	database := lookupEnvOrGetDefault("NEO4J_DATABASE", "")
	if !strings.HasPrefix(lookupEnvOrGetDefault("NEO4J_VERSION", ""), "4") {
		database = ""
	}
	return &Neo4jConfiguration{
		Url:      lookupEnvOrGetDefault("NEO4J_URI", ""),
		Username: lookupEnvOrGetDefault("NEO4J_USER", ""),
		Password: lookupEnvOrGetDefault("NEO4J_PASSWORD", ""),
		Database: database,
	}
}

func lookupEnvOrGetDefault(key string, defaultValue string) string {
	if env, found := os.LookupEnv(key); !found {
		return defaultValue
	} else {
		return env
	}
}

func unsafeClose(closeable io.Closer) {
	if err := closeable.Close(); err != nil {
		log.Fatal(fmt.Errorf("could not close resource: %w", err))
	}
}