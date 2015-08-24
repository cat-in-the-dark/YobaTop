package yobaludum

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/binding"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

// PlayerInfo is API object with all info from game
type PlayerInfo struct {
	Name string `json:"name" binding:"required"`
	Time int    `json:"time" binding:"required"`
}

// PlayerData is dataStore object
type PlayerData struct {
	Name        string
	Time        int
	CreatedAt   time.Time
	Country     string
	Region      string
	City        string
	CityLatLong string
	IP          string
}

func appEngine(c martini.Context, r *http.Request) {
	c.Map(appengine.NewContext(r))
}

func init() {
	m := martini.Classic()
	m.Use(appEngine)
	m.Use(martini.Logger())
	m.Get("/", func(c context.Context, res http.ResponseWriter) {
		players, err := getAllPlayers(c)
		if err != nil {
			log.Errorf(c, "Can't load players from DS: %v", err)
		}
		tmplt.Execute(res, players)
	})

	m.Get("/players.json", func(c context.Context, res http.ResponseWriter) {
		if players, err := getAllPlayers(c); err != nil {
			log.Errorf(c, "Can't load players from DS: %v", err)
		} else {
			if b, err := json.Marshal(players); err != nil {
				log.Errorf(c, "Can't convert players to json: %v", err)
			} else {
				res.Write(b)
				return
			}
		}
		res.WriteHeader(404)
	})
	m.Get("/results.json", func(c context.Context, res http.ResponseWriter) {
		if players, err := getAllResults(c); err != nil {
			log.Errorf(c, "Can't load results from DS: %v", err)
		} else {
			if b, err := json.Marshal(players); err != nil {
				log.Errorf(c, "Can't convert results to json: %v", err)
			} else {
				res.Write(b)
				return
			}
		}
		res.WriteHeader(404)
	})
	m.Post("/", binding.Bind(PlayerInfo{}), handlePlayer)

	http.Handle("/", m)
}

func save(c context.Context, data *PlayerData) (err error) {
	id := data.Name + data.IP
	data.CreatedAt = time.Now()
	key := datastore.NewKey(c, "Players", id, 0, nil)
	if oldPlayer, er := findPlayer(c, key); er != nil {
		log.Infof(c, "Saving Player[%s]=%v", id, data)
		_, err = datastore.Put(c, key, data)
	} else {
		if oldPlayer.Time > data.Time {
			log.Infof(c, "Updating Player[%s]=%v", id, data)
			_, err = datastore.Put(c, key, data)
		}
	}

	keyResults := datastore.NewIncompleteKey(c, "Results", nil)
	datastore.Put(c, keyResults, data)

	return
}

func findPlayer(c context.Context, key *datastore.Key) (*PlayerData, error) {
	data := new(PlayerData)
	if err := datastore.Get(c, key, data); err != nil {
		return new(PlayerData), err
	}
	return data, nil
}

func getAllPlayers(c context.Context) ([]PlayerData, error) {
	q := datastore.NewQuery("Players").Order("Time").Limit(200)
	players := make([]PlayerData, 0, 200)
	_, err := q.GetAll(c, &players)
	return players, err
}

func getAllResults(c context.Context) ([]PlayerData, error) {
	q := datastore.NewQuery("Results").Order("Time").Limit(1000)
	players := make([]PlayerData, 0, 1000)
	_, err := q.GetAll(c, &players)
	return players, err
}

func handlePlayer(c context.Context, data PlayerInfo, w http.ResponseWriter, r *http.Request) {
	player := new(PlayerData)
	player.Country = r.Header.Get("X-AppEngine-Country")
	player.Region = r.Header.Get("X-AppEngine-Region")
	player.City = r.Header.Get("X-AppEngine-City")
	player.CityLatLong = r.Header.Get("X-Appengine-Citylatlong")
	player.Name = data.Name
	player.Time = data.Time
	player.IP = strings.Split(r.RemoteAddr, ":")[0]
	log.Infof(c, "RECEIVE: %v", data)
	log.Infof(c, "SAVE: %v", player)

	if err := save(c, player); err != nil {
		log.Errorf(c, "Can't save player[%v] in DS: %v", player, err)
	}
}

var tmplt = template.Must(template.New("players").Parse(`
<html>
  <head>
    <title>YoBA highscores</title>
  </head>
  <body>
		<table>
			<thead></thead>
				<tr>
					<th>Nickname</th>
					<th>Score</th>
					<th>Country</th>
				</tr>
			<tbody>
				{{range .}}
					<tr>
						<td>{{.Name}}</td>
						<td>{{.Time}}</td>
						<td>{{.Country}}</td>
					</tr>
				{{end}}
			</tbody>
		</table>
  </body>
</html>
`))
