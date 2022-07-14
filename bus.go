package main

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	Id       int
	Name     string
	Password string
}

type Seat struct {
	Id      int
	Status  bool
	Bus_id  int
	User_id int
}

type Bus struct {
	Id          int
	Start_city  string
	Destination string
	Launch_time string
}

type SelectStationData struct {
	Buses    []Bus
	Stations []string
}

type Ticket struct {
	Id          int
	BusNumber   int
	SeatNumber  int
	Station     string
	Destination string
	Time        string
}

var db *sql.DB
var err error
var ticketNumber = 1

func checkStatusOfSeat(Bus_id int, Seat_id int) bool {
	statusOfSeat, err := db.Query("SELECT status FROM Seat WHERE Bus_id = ? AND id=?;", Bus_id, Seat_id)
	if err != nil {
		return false
	}
	if statusOfSeat.Next() {
		var res int
		statusOfSeat.Scan(&res)
		defer statusOfSeat.Close()
		return res == 0
	} else {
		return false
	}
}
func cancelReceive(Bus_id int, Seat_id int) bool {
	if checkStatusOfSeat(Bus_id, Seat_id) {
		update, err := db.Query("UPDATE Seat SET status=1 WHERE id=? AND Bus_id=?", Seat_id, Bus_id)
		if err != nil {
			return false
		}
		defer update.Close()
		return true
	} else {
		return false
	}
}
func getAllStations() []string {
	state, err := db.Query("SELECT distinct start_city FROM bus union select distinct destination from bus;")
	if err != nil {
		panic(err.Error())
	}
	var cities []string
	var data string
	for state.Next() {
		err := state.Scan(&data)
		if err != nil {
			panic(err)
		}
		cities = append(cities, data)
	}
	defer state.Close()
	return cities
}
func hasAvailableSeats(Bus_id int) bool {
	checkAllSeats, err := db.Query("SELECT * FROM Seat WHERE status = 1 AND Bus_id = ?;", Bus_id)
	if err != nil {
		return false
	}
	defer checkAllSeats.Close()
	return checkAllSeats.Next()
}

func getAllBuses(start_city string, destination string) []Bus {
	checkAllBuses, err := db.Query("SELECT * FROM Bus WHERE start_city = ? AND destination = ?;", start_city, destination)
	if err != nil {
		panic(err)
	}
	defer checkAllBuses.Close()
	var singleBus Bus
	var buses []Bus
	for checkAllBuses.Next() {
		err = checkAllBuses.Scan(&singleBus.Id, &singleBus.Start_city, &singleBus.Destination, &singleBus.Launch_time)
		if err != nil {
			panic(err)
		}
		if hasAvailableSeats(singleBus.Id) {
			buses = append(buses, singleBus)
		}
	}
	return buses
}
func index(response http.ResponseWriter, request *http.Request) {
	tmp, _ := template.ParseFiles("index.html")
	tmp.Execute(response, nil)
}

func logIn(response http.ResponseWriter, request *http.Request) {
	request.ParseForm()
	userName := request.Form.Get("username")
	passWord := request.Form.Get("password")
	if isUserexistInDataBase(userName, passWord) {
		http.Redirect(response, request, "/home", http.StatusSeeOther)
	} else {
		http.Redirect(response, request, "/index", http.StatusSeeOther)
		//	http.Error(response, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	}
}
func isUserexistInDataBase(name string, password string) bool {
	var userlogin User
	err = db.QueryRow("SELECT * FROM User WHERE name = ? AND password = ?;", name, password).Scan(&userlogin.Id, &userlogin.Name, &userlogin.Password)
	//if err = nul then there is a user with this password in our database
	return err == nil
}
func getFirstAvailableSeat(Bus_id int) int {
	var seat Seat
	err := db.QueryRow("SELECT * FROM Seat where status=1 AND bus_id=? Limit 1", Bus_id).Scan(&seat.Id, &seat.Status, &seat.Bus_id, &seat.User_id)
	if err != nil {
		panic(err)
	}
	return seat.Id
}
func selctBusForm(response http.ResponseWriter, request *http.Request) {
	request.ParseForm()
	Station := request.Form.Get("Station")
	Destination := request.Form.Get("Destination")
	var data SelectStationData
	data.Buses = getAllBuses(Station, Destination)
	data.Stations = getAllStations()
	tmp, _ := template.ParseFiles("select-bus.html")
	tmp.Execute(response, data)
}

func book(response http.ResponseWriter, request *http.Request) {
	request.ParseForm()
	var ticket Ticket
	ticket.Id = ticketNumber
	ticketNumber++
	ticket.Station = request.Form.Get("Station")
	ticket.Destination = request.Form.Get("Destination")
	intVar, _ := strconv.Atoi(request.Form.Get("BusNumber"))
	ticket.BusNumber = intVar
	ticket.Time = request.Form.Get("Time")
	ticket.SeatNumber = getFirstAvailableSeat(ticket.BusNumber)
	tmp, _ := template.ParseFiles("ticket.html")
	tmp.Execute(response, ticket)
}

func reserveSeat(Bus_id int, id int) bool {
	update, err := db.Query("UPDATE Seat SET status=0 WHERE id=? AND Bus_id=?", id, Bus_id)
	if err != nil {
		panic(err.Error())
	}
	defer update.Close()
	return err != nil
}

func confirmBooking(response http.ResponseWriter, request *http.Request) {
	request.ParseForm()
	busNumber, _ := strconv.Atoi(request.Form.Get("Bus-Number"))
	seatNumber, _ := strconv.Atoi(request.Form.Get("Seat-Number"))

	reserveSeat(busNumber, seatNumber)
	http.Redirect(response, request, "/home", http.StatusSeeOther)
}

func home(response http.ResponseWriter, request *http.Request) {
	tmp, _ := template.ParseFiles("home.html")
	tmp.Execute(response, nil)
}

func cancel(response http.ResponseWriter, request *http.Request) {
	tmp, _ := template.ParseFiles("Cancel.html")
	tmp.Execute(response, nil)
}

func confirmCancellation(response http.ResponseWriter, request *http.Request) {
	request.ParseForm()
	busNumber, _ := strconv.Atoi(request.Form.Get("Bus-Number"))
	seatNumber, _ := strconv.Atoi(request.Form.Get("Seat-Number"))
	flag := cancelReceive(busNumber, seatNumber)
	if flag {
		http.Redirect(response, request, "/home", http.StatusSeeOther)
	} else {
		http.Redirect(response, request, "/cancel", http.StatusSeeOther)
	}
}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	pswd := "1234"
	//db, err = sql.Open("mysql", "root:"+pswd+"@tcp(localhost:3306)/bookbus")
	db, err = sql.Open("mysql", "master:"+pswd+"@tcp(192.168.43.71:3306)/bookbus")
	if err != nil {
		panic(err.Error())
	}
	http.HandleFunc("/index", index)
	http.HandleFunc("/confirm-cancellation", confirmCancellation)
	http.HandleFunc("/select-bus", selctBusForm)
	http.HandleFunc("/home", home)
	http.HandleFunc("/cancel", cancel)
	http.HandleFunc("/login", logIn)
	http.HandleFunc("/book", book)
	http.HandleFunc("/confirm", confirmBooking)
	http.ListenAndServe(":8080", nil)
	db.Close()
}
