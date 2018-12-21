package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
	resty "gopkg.in/resty.v1"
)

const userAgent = "toggl-export"

type ProjectTitle struct {
	Project string `json:"project"`
}

type TimeEntryTitle struct {
	Name string `json:"time_entry"`
}

type TimeEntry struct {
	Title *TimeEntryTitle `json:"title"`
	Time  int             `json:"time"`
}

type ProjectEntry struct {
	Title       *ProjectTitle `json:"title"`
	TimeEntries []*TimeEntry  `json:"items"`
}

type Response struct {
	ProjectEnties []*ProjectEntry `json:"data"`
}

type Config struct {
	ApiToken    string `toml:"api_token"`
	WorkspaceID string `toml:"workspace_id"`
	UserName    string `toml:"user_name"`
}

func main() {
	startDate := flag.String("start", "", "The first day to start the report from")
	endDate := flag.String("end", "", "The last day to export from")

	flag.Parse()

	var config Config
	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		panic(err)
	}

	resp, _ := resty.
		R().
		SetHeader("Accept", "application/json").
		SetBasicAuth(config.ApiToken, "api_token").
		SetQueryParams(map[string]string{
			"workspace_id": config.WorkspaceID,
			"since":        *startDate,
			"until":        *endDate,
			"user_agent":   userAgent,
		}).
		Get("https://toggl.com/reports/api/v2/summary")

	var responseBody Response
	err := json.Unmarshal(resp.Body(), &responseBody)

	if err != nil {
		panic(err)
	}

	data := make([][]string, 0)
	var total float64

	for _, projectEntry := range responseBody.ProjectEnties {
		for _, timeEntry := range projectEntry.TimeEntries {
			hours := float64(timeEntry.Time) / 3600000
			roundedHours := math.Round(hours*4) / 4

			if roundedHours == 0 {
				roundedHours = 0.25
			}

			total += roundedHours
			data = append(data, []string{config.UserName, projectEntry.Title.Project, "Production", timeEntry.Title.Name, strconv.FormatFloat(roundedHours, 'f', 2, 64)})
		}
	}

	file, err := os.Create("result.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	w := csv.NewWriter(file)
	w.WriteAll(data)

	if err := w.Error(); err != nil {
		log.Fatalln("error writing csv:", err)
	}

	fmt.Printf("Total time: %vh\n", total)
}
