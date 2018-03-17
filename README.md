# Marathon-Tools API

This is the backend for Marathon-Tools. It provides a timer, run managing, showing donation counts and automatic social media management. I decided to write this because I didn't really like the other options out there. It will be used first used live in production at GSM18/Speedcon.

## Getting Started
Currently there are is no documentation on the API Endpoints or on the data the websocket sends. You can find all endpoints in `main.go` and all websocket data the websocket sends in `api/common/wsUpdates.go`. I will eventually write docs at a later point.

### Development

The API comes with a docker compose to get you up and running quickly.

You need a valid twitch and twitter tokens, otherwise the social functions won't work.

* You can get TWITCH_CLIENT_ID, TWITCH_CLIENT_SECRET, TWITCH_CALLBACK, TWITTER_KEY, TWITTER_SECRET and TWITTER_CALLBACK from the respective pages after having created the application. There is also a [web frontend](https://github.com/onestay/MarathonTools-Client) in existence which can handle the callbacks from twitch and twitter.
* MARATHON_SLUG is used for donation info and will used by the DonationProvider. Currently the only donation provider is speedrun.com however I plan on adding more in the future.
* REFRESH_INTERVAL is the interval in which the timer will send out time updates via the websocket
* HTTP_PORT is the port for the webserver to listen on

All you have to do is 

```
git clone https://github.com/onestay/MarathonTools-API
cd MarathonTools-API
```
Rename docker-compose.example.yml to docker-compose.yml and fill out env 
vars.

Create a runs.json in config folder. Take a look at runs_gsm18.json for an example. Currently the api will crash if there are no runs. This will be fixed in an later update.
```
docker-compose up
```


This will also automatically start a mongodb and redis instance which is needed to run. The API will automatically rebuild when you change the source code.

## Deployment

When deploying I would reccomend using Docker and starting up the containers individually since with the docker-compose the source code get's added to the container as a volume and will rebuild once changed.

Create a new docker network and 2 volumes for mongo and redis. Refer to the docker documentation for these two to read how to store data on on those volumes.

Next start up the redis and mongo instances attached to the perviously created network. It is important that those two are started before the api is started. 

Next build the api container and start it also attached to the network with all the relevant env vars filled out.

## Built With

* [gorilla/websocket](https://github.com/gorilla/websocket) - The websocket server
* [go-redis/redis](https://github.com/go-redis/redis) - For redis communication
* [go-mgo/mgo](https://github.com/go-mgo/mgo/) - For mongo communication
* [httprouter](https://github.com/julienschmidt/httprouter) - The best go router
* [dghubble/oauth1](https://github.com/dghubble/oauth1) - For Twitter oauth stuff

## Contributing

There's still a lot to be done here. I plan on adding a bunch of more features and stuff. If you just want to fix a little bug or refactor some code go ahead. If you plan on adding a major feature please contact me first.

## Authors

* **Onestay** - *Initial work* - [Onestay](https://github.com/onestay)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details

