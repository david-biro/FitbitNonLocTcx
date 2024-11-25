package main

import (
	"FitbitNonLocTcx/data"
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/fitbit"
)

var (
	codeVerifier  string            // A cryptographically secure random value.
	codeChallenge string            // A base64-encoded SHA-256 transformation of the Code Verifier.
	done          = make(chan bool) // Channel to signal when the server should stop.
	server        *http.Server      // HTTP server to handle redirect.
	stateAuth     string            // A unique value generated by the app in authorization URL.
	stateRedir    string            // A unique value passed back from server in redirect request and validated by the app if it matches with the one in authorization URL.
	token         string            // Access token to request user data.
)

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	jsonFile, err := os.Open("credentials.json")
	handleError(err)
	defer jsonFile.Close()
	ouathCfg, err := readCredFile(jsonFile)
	handleError(err)
	codeVerifier, err = generateCodeVerifier(43)
	handleError(err)
	codeChallenge, err = generateCodeChallenge(codeVerifier)
	handleError(err)

	http.HandleFunc("/callback", handleOAuth2Callback)
	http.HandleFunc("/token-received", handleTokenReceived)

	server = &http.Server{Addr: ":8080"}

	// Generate and print the authorization URL
	authURL := getAuthURL(codeChallenge, ouathCfg)

	// Open the URL in the default browser
	err = openBrowser(authURL)
	if err != nil {
		log.Fatalf("Error opening browser: %v", err)
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()

	// Wait for the server to finish
	<-done
	fmt.Println("Server stopped gracefully")
}

// Reads the credentials.json file
func readCredFile(reader io.Reader) (*oauth2.Config, error) {
	var apiCred data.Credentials // Fitbit API credentials: OAuth 2.0 Client ID (and Client Secret in case of Application Type: Server)

	// Read the file's content
	byteValue, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %s", err)
	}

	// Unmarshal the JSON data into a struct
	json.Unmarshal(byteValue, &apiCred)
	if err := json.Unmarshal(byteValue, &apiCred); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %s", err)
	}

	if (apiCred.CId != "") && (apiCred.RedirectURL != "") {
		// OAuth2 Config setup
		return &oauth2.Config{
			ClientID:     apiCred.CId,
			ClientSecret: apiCred.CSecret,
			RedirectURL:  apiCred.RedirectURL,
			Scopes:       []string{"activity", "heartrate", "location", "profile"}, // only request what is really needed
			//"activity", "cardio_fitness", "electrocardiogram", "heartrate", "location", "nutrition", "oxygen_saturation", "profile", "respiratory_rate", "settings", "sleep", "social", "temperature", "weight"
			Endpoint: fitbit.Endpoint,
		}, nil
	} else {
		err := "The clientID and redirect URL cannot be empty."
		return nil, fmt.Errorf("ERROR %s", err)
	}
}

// Generates a code challenge from the code verifier
func generateCodeChallenge(codeVerifier string) (string, error) {
	if codeVerifier == "" {
		return "", fmt.Errorf("error: empty codeVerifier string")
	}
	hash := sha256.Sum256([]byte(codeVerifier))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:]), nil
}

// Generates a random code verifier (need to change to use unreserved characters, RFC3986)
/*func generateCodeVerifier() (string, error) {

	cv := make([]byte, 43) // verifier length = 43
	_, err := rand.Read(cv)
	if err != nil {
		log.Fatalf("Failed to generate random bytes: %v", err)
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(cv), nil
}*/

// Generates a random code verifier accroding to RFC 7636 RFC3986
func generateCodeVerifier(length int) (string, error) {
	const rfc3986Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	if length < 43 || length > 128 {
		return "", fmt.Errorf("code verifier length must be between 43 and 128 characters")
	}

	verifier := make([]byte, length)
	for i := range verifier {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(rfc3986Chars))))
		if err != nil {
			return "", err
		}
		verifier[i] = rfc3986Chars[index.Int64()]
	}
	return string(verifier), nil
}

// Generates a random 32 character length string for "state", https://go.dev/play/p/Lwnd5B7VYIL
func generateRandomString() string {
	charset := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, 32)
	max := big.NewInt(int64(len(charset)))

	for i := 0; i < 32; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		result[i] = charset[n.Int64()]
	}
	stateAuth = string(result) // Store it for later comparison
	return stateAuth
}

// Generates the authorization URL
func getAuthURL(codeChallenge string, ouathCfg *oauth2.Config) string {
	return fmt.Sprintf(
		"https://www.fitbit.com/oauth2/authorize?response_type=token&client_id=%s&redirect_uri=%s&scope=%s&code_challenge=%s&code_challenge_method=%s&state=%s",
		ouathCfg.ClientID, ouathCfg.RedirectURL, scopeStringBuilder(ouathCfg.Scopes), codeChallenge, "S256", generateRandomString())
}

// Concatenates the scopes with a "+"
func scopeStringBuilder(scopes []string) string {
	var scopeString = ""
	for _, s := range scopes {
		scopeString = scopeString + s + "+"
	}
	if lastChar := len(scopeString) - 1; lastChar >= 0 && scopeString[lastChar] == '+' {
		scopeString = scopeString[:lastChar]
	}
	return scopeString
}

// Opens a URL in the default browser
func openBrowser(url string) error {
	switch {
	case strings.Contains(strings.ToLower(os.Getenv("OS")), "windows"):
		cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		return cmd.Start()
	case strings.Contains(strings.ToLower(os.Getenv("OS")), "darwin"):
		cmd := exec.Command("open", url)
		return cmd.Start()
	default:
		cmd := exec.Command("xdg-open", url)
		return cmd.Start()
	}
}

// Handles the OAuth2 callback
func handleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	// Construct a simple HTML page with JavaScript to extract the access token
	html := `
	<html>
		<body>
			<script type="text/javascript">
				var fragmentString = window.location.hash.substr(1);
				var params = {};
				fragmentString.split("&").forEach(function (pair) {
					var keyValue = pair.split("=");
					params[keyValue[0]] = keyValue[1];
				});
				var accessToken = params["access_token"];
				var state = params["state"];
				if (accessToken && state) {
					fetch('/token-received?token=' + accessToken + '&state=' + state)
						.then(response => response.text())
						.then(data => {
							document.write("Access token and state received and sent to server.");
						});
				} else {
					document.write("Error: Access token or state not found in the URL fragment.");
				}
			</script>
		</body>
		</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// Handles the token reception
func handleTokenReceived(w http.ResponseWriter, r *http.Request) {
	token = r.URL.Query().Get("token")
	stateRedir = r.URL.Query().Get("state")
	if token != "" {
		fmt.Println("Access Token:", token)
		w.Write([]byte("Token received and printed to the server console."))
		if strings.Compare(stateAuth, stateRedir) == 0 {
			w.Write([]byte("State matches with the one sent in auth URL."))
			fetchActivityData(os.Args)
		} else {
			w.Write([]byte("The redirect request not originated from this app."))
		}
	} else {
		w.Write([]byte("No token received."))
	}
}

// Fetches activity data using the access token, JSON
func fetchActivityData(args []string) {
	fmt.Println("Fetching activity data...")

	if len(args) == 2 {

		url := "https://api.fitbit.com/1/user/-/activities/date/" + args[1] + ".json"

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Add("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Failed to fetch activity data: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Failed to read response body: %v", err)
		}

		var prettyJson bytes.Buffer
		error := json.Indent(&prettyJson, body, "", "\t")
		if error != nil {
			log.Println("JSON parse error: ", error)
			return
		}
		fmt.Println("Activity Data:", prettyJson.String())

		// Unmarshal the JSON into the Activities struct
		var activities data.Activities
		err = json.Unmarshal(body, &activities)
		if err != nil {
			log.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		// Display the list of activities with their index
		fmt.Println("Available Activities:")
		for i, activity := range activities.Activities {
			fmt.Printf("ID: %d\n", i+1)
			fmt.Printf("Activity Name: %s\n", activity.Name)
			fmt.Printf("Distance: %.2f\n", activity.Distance)
			fmt.Printf("Start date: %s\n", activity.StartDate+" "+activity.StartTime)
			fmt.Println("-------------")
		}

		// Prompt the user to choose an activity
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter the number of the activity you want to choose: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read input: %v", err)
		}

		input = strings.TrimSpace(input)
		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(activities.Activities) {
			fmt.Println("Invalid choice. Please enter a valid number.")
			return
		}

		chosenActivity := activities.Activities[choice-1]
		fmt.Println("You selected: " + strconv.Itoa(choice) + " " + chosenActivity.ActivityParentName + " " + chosenActivity.StartDate + " " + chosenActivity.StartTime)
		fileNameToSave := chosenActivity.ActivityParentName + "-" + strconv.FormatInt(chosenActivity.LogID, 10)

		// for debug purposes save all activity on that day
		// saveToFile("All-"+args[1]+".json", prettyJson.Bytes())

		xml := getActivityTcx(chosenActivity.LogID)

		injectActivityTcx(fileNameToSave, xml, chosenActivity.ActivityParentName, time.Duration(chosenActivity.Duration/1000)*time.Second,
			strconv.FormatFloat(chosenActivity.Distance*1000.0, 'f', -1, 64), strconv.Itoa(chosenActivity.Calories))
		// FormatFloat(f: output fixed point, -1: precision automatically det, 64: input is float 64)

	} else if len(args) < 2 {
		log.Fatalf("No date specified. Give a date in a format YYYY-MM-DD!")
	} else {
		log.Fatalf("Maximum of one date can be given in a format YYYY-MM-DD.")
	}

}

// Dumps the "data" byte slice into a file
func saveToFile(fileName string, data []byte) {
	directory := filepath.Dir(fileName)
	err := os.MkdirAll(directory, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		log.Fatalf("Failed to create directory: %v", err)
	}

	err = os.WriteFile(filepath.Join(directory, fileName), data, os.FileMode(0644))
	if err != nil {
		log.Fatalf("Failed to save data to '%s': %v", fileName, err)
	}

	fmt.Println("Data saved to", fileName)
}

// Gets the selected activity in tcx, based on its logId (activities : logId)
func getActivityTcx(logId int64) *etree.Document {
	url := "https://api.fitbit.com/1/user/-/activities/" + strconv.FormatInt(logId, 10) + ".tcx?includePartialTCX=true"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Add("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to fetch activity data: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromString(string(body)); err != nil {
		log.Fatalf("Failed to parse XML: %v", err)
	}
	return doc
}

// Modifies the acquired tcx file
func injectActivityTcx(fName string, xmlDoc *etree.Document, actName string, totalTime time.Duration, distMeters string, calories string) {

	// modify TCX in case Swim, create trackPtElementStart as start and trackPtElementEnd as end point
	if actName == "Swim" {
		// Navigate to the root element
		root := xmlDoc.SelectElement("TrainingCenterDatabase").SelectElement("Activities").SelectElement("Activity")
		root.SelectAttr("Sport").Value = actName
		idElement := string(root.SelectElement("Id").Text())
		nameElement := etree.NewElement("Name")
		nameElement.SetText("Fitbit")
		creatorElement := root.SelectElement("Creator")
		creatorElement.AddChild(nameElement)
		lapElement := root.CreateElement("Lap")

		tss, _ := convertTimestamp(idElement, 0) // Convert start timestamp
		lapElement.CreateAttr("StartTime", tss)

		totalTimeSecondsElement := etree.NewElement("TotalTimeSeconds")
		totalTimeSecondsElement.SetText(strconv.FormatFloat(totalTime.Seconds(), 'f', -1, 64))
		lapElement.AddChild(totalTimeSecondsElement)
		lapElement.CreateElement("DistanceMeters").SetText(distMeters)
		lapElement.CreateElement("Calories").SetText(calories)
		lapElement.CreateElement("Intensity").SetText("Active")
		lapElement.CreateElement("TriggerMethod").SetText("Manual")
		trackElement := etree.NewElement("Track")
		lapElement.AddChild(trackElement)
		// Start point
		trackPtElementStart := etree.NewElement("Trackpoint")
		trackElement.AddChild(trackPtElementStart)
		timeElementStart := etree.NewElement("Time")
		timeElementStart.SetText(tss)
		trackPtElementStart.AddChild(timeElementStart)
		distMetElementStart := etree.NewElement("DistanceMeters")
		distMetElementStart.SetText("0")
		trackPtElementStart.AddChild(distMetElementStart)
		// End point
		trackPtElementEnd := etree.NewElement("Trackpoint")
		trackElement.AddChild(trackPtElementEnd)
		timeElementEnd := etree.NewElement("Time")

		tse, _ := convertTimestamp(idElement, totalTime) // Convert end timestamp
		timeElementEnd.SetText(tse)
		trackPtElementEnd.AddChild(timeElementEnd)
		distMetElementEnd := etree.NewElement("DistanceMeters")
		distMetElementEnd.SetText(distMeters)
		trackPtElementEnd.AddChild(distMetElementEnd)
	}

	// modify TCX in case Treadmill or Weights, add device name
	if (actName == "Treadmill") || (actName == "Weights") {
		// Navigate to the root element
		root := xmlDoc.SelectElement("TrainingCenterDatabase").SelectElement("Activities").SelectElement("Activity").SelectElement("Creator")
		nameElement := etree.NewElement("Name")
		nameElement.SetText("Fitbit")
		root.AddChild(nameElement)
	}

	xmlDoc.Indent(2)
	xmlString, err := xmlDoc.WriteToString()
	if err != nil {
		log.Fatalf("Failed to write XML to string: %v", err)
	}
	fmt.Println(string(xmlString))
	saveToFile(fName+".tcx", []byte(xmlString))
	// Shut down server
	go func() {
		if err := server.Shutdown(context.Background()); err != nil {
			log.Fatalf("Server Shutdown Failed:%+v", err)
		}
		done <- true
	}()
}

// Converts the timestamp from RFC3339 to UTC
func convertTimestamp(timeStamp string, addSecond time.Duration) (string, error) {
	t, err := time.Parse(time.RFC3339, timeStamp)
	if err != nil {
		return "", err
	}
	utcTime := t.UTC()
	newTime := utcTime.Add(addSecond)
	return newTime.Format(time.RFC3339), nil
}
