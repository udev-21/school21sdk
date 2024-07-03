package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// func greet(w http.ResponseWriter, r *http.Request) {
// 	// post request to auth server
// 	// To get an access token (JWT) you need to make a POST request to the Access Token URL (https://auth.sberclass.ru/auth/realms/EduPowerKeycloak/protocol/openid-connect/token).
// 	// In the request body parameters you need to specify the values for parameters - "username" (=$login), "password" (=$password), "grant_type" (="password") and "client_id" (="s21-open-api")

// 	w.WriteHeader(response.StatusCode)
// 	w.Write([]byte("Hello, " + login))

// }

type jwt struct {
	AccessToken string `json:"access_token"`
}

var jwt_token *jwt = nil

var sdk_campus_id = "667a42af-5469-4a33-9858-677d9d20956a"

var participants map[string]int

var colaitions []string = []string{"438", "437", "436", "435"}

func main() {
	auth()
	getParticipants()
	ticker := time.NewTicker(30 * time.Minute)
	go func() {
		for range ticker.C {
			fmt.Println("Updating...")
			auth()
			calculatePoints()
			// git add readme.md
			// git commit -m "update leaderboard"
			// git push
			addCmd := exec.Command("git", "add", "readme.md")
			if output, err := addCmd.CombinedOutput(); err != nil {
				fmt.Printf("Error adding README.md: %s\n", err)
				fmt.Printf("Output: %s\n", string(output))
				panic(err)
			}

			// Step 2: Commit the changes with the message "updated leaderboard"
			commitCmd := exec.Command("git", "commit", "-m", "updated leaderboard")
			if output, err := commitCmd.CombinedOutput(); err != nil {
				fmt.Printf("Error committing changes: %s\n", err)
				fmt.Printf("Output: %s\n", string(output))
				panic(err)
			}

			// Step 3: Push the changes to the remote repository
			pushCmd := exec.Command("git", "push")
			if output, err := pushCmd.CombinedOutput(); err != nil {
				fmt.Printf("Error pushing changes: %s\n", err)
				fmt.Printf("Output: %s\n", string(output))
				panic(err)
			}
		}
	}()
	select {}

}

func auth() {
	login := "donnettp@student.21-school.ru"
	password := os.Getenv("PASSWORD")
	grant_type := "password"
	client_id := "s21-open-api"
	response, err := http.Post("https://auth.sberclass.ru/auth/realms/EduPowerKeycloak/protocol/openid-connect/token", "application/x-www-form-urlencoded", strings.NewReader("username="+login+"&password="+password+"&grant_type="+grant_type+"&client_id="+client_id))

	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	// print body to console
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(body, &jwt_token)
	if err != nil {
		panic(err)
	}
}

func getParticipants() {
	if jwt_token == nil {
		auth()
	}
	participants = make(map[string]int)

	for _, colaition := range colaitions {
		// https://edu-api.21-school.ru/services/21-school/api/v1/campus/{campus_id}/participants
		req, err := http.NewRequest("GET", "https://edu-api.21-school.ru/services/21-school/api/v1/coalitions/"+colaition+"/participants?limit=1000", nil)
		if err != nil {
			panic(err)
		}

		// set headers
		req.Header.Set("Authorization", "Bearer "+jwt_token.AccessToken)

		// send request
		client := &http.Client{}
		resp, err := client.Do(req)

		if err != nil {
			panic(err)
		}

		// print body to console
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		type Res struct {
			Participants []string `json:"participants"`
		}
		var res Res
		err = json.Unmarshal(body, &res)
		if err != nil {
			panic(err)
		}

		for _, login := range res.Participants {
			participants[login] = 0
			// fmt.Er(login)
		}
		time.Sleep(1 * time.Second)
	}
}

func calculatePoints() {
	// https://edu-api.21-school.ru/services/21-school/api/v1/participants/{login}/points

	for login := range participants {
		points := getPeerPointofUser(login)
		participants[login] = points

		time.Sleep(1 * time.Second)
	}

	// sort map by value
	var byPoint map[int][]string = make(map[int][]string)

	for k, v := range participants {
		byPoint[v] = append(byPoint[v], k)
	}

	keys := make([]int, 0, len(byPoint))
	for k := range byPoint {
		keys = append(keys, k)
	}
	// sort keys
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] < keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	// save to file sorted map
	f, err := os.OpenFile("readme.md", os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, err = f.WriteString("# Leaderboard by peer points\n\n")
	if err != nil {
		panic(err)
	}
	_, err = f.WriteString("### Updated at: " + time.Now().UTC().Add(5*time.Hour).Format("2006-01-02 15:04:05") + "\n\n")
	if err != nil {
		panic(err)
	}

	_, err = f.WriteString("| № | Login | Points |\n|---|-------|--------|\n")
	if err != nil {
		panic(err)
	}

	ic := 1
	// print sorted map
	for _, k := range keys {
		for _, login := range byPoint[k] {
			f.WriteString(fmt.Sprintf("|%d|%s|%d|\n", ic, login, participants[login]))
			ic++
		}
	}
}

func getPeerPointofUser(login string) int {

	req, err := http.NewRequest("GET", "https://edu-api.21-school.ru/services/21-school/api/v1/participants/"+login+"/points", nil)
	if err != nil {
		panic(err)
	}

	// set headers
	req.Header.Set("Authorization", "Bearer "+jwt_token.AccessToken)

	// send request
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	if resp.StatusCode == 404 {
		return 0
	} else if resp.StatusCode != 200 {
		// print body to console
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(body))
		panic("Error: ")
	}

	type Points struct {
		PeerReviewPoints int `json:"peerReviewPoints"`
		CodeReviewPoints int `json:"codeReviewPoints"`
		Coins            int `json:"coins"`
	}
	var points Points
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(body, &points)
	if err != nil {
		panic(err)
	}

	return points.PeerReviewPoints
}
