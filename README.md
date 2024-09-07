 # Fitbit activity downloader
 Downloads and converts non-GPS activities (i.e., activities where location data was not logged) or activities that cannot be synced to Strava from your Fitbit account, and converts them to Garmin's Training Center Database XML (TCX) files, making it possible to manually upload them to Strava. 

 # Configure the app
 Fitbit requires the use of the OAuth 2.0 Authorization Framework, so as a first step, [register an app](https://dev.fitbit.com/apps/new) at Fitbit Developer portal, to obtain OAuth 2.0 Client ID. Place the Client ID and Redirect URL in the credentials.json file:
 ```
{
    "clientID": "",
    "clientSecret": "",
    "redirectUrl": "http://localhost:8080/callback"
}
```

This app cannot securely store the Client Secret in client-side code, so it is not being used.


```
FitbitNonLocTcx
├── data                    
│   └── data.go             # Data structures 
├── credentials.json        # Fitbit credentials
├── go.mod                  
├── go.sum                  
├── main.go
├── main_test.go
└── README.md
```

 # Using the app

 The activities can be obtained by specifying a date: ```go run main.go <date-of-activities> ```

 Example:  
 ```
 go run main.go 2024-08-11
 go run main.go today
 ```

 The first time, a browser window will pop up asking you to log in to your Fitbit account, and it will then display Fitbit's authorization webpage. After granting permissions, you can close the browser window. Then, on the console, select the activity you want to save in TCX format.

 # References
 - [RFC6749, The OAuth 2.0 Authorization Framework](https://datatracker.ietf.org/doc/html/rfc6749)
 - [dev.fitbit.com](https://dev.fitbit.com/build/reference/)

 # Contributing
 Feedbacks and recommendations are welcomed.

 # Licensing
 This project is licensed under the GNU GPLv3 License - see the LICENSE file for details.
