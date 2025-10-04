package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

type Rfid_card struct {
	Student_name string
	Rfid         string `json:"RFID"`
	Created_at   time.Time
	LastIn       time.Time
	LastOut      time.Time
	National_id  string
	Is_present   int8
}

// Database connection
var db *sql.DB
var form_card Rfid_card

func init() {
	var err error
	db, err = sql.Open("sqlite3", "./DB/RFID.db")
	if err != nil {
		panic(err)
	}
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections by default
	},
}

func newStudent(cardUID string) error {
	form_card.Rfid = cardUID
	form_card.Created_at = time.Now()
	form_card.LastIn = time.Now()
	form_card.LastOut = time.Now()
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Student's Name: ")
	input, err := reader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("error reading student's name: %w", err)
	} else {
		inputstr := string(input)
		form_card.Student_name = strings.TrimSpace(inputstr)
	}

	fmt.Print("Student's National ID: ")
	input, err = reader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("error reading student's national ID: %w", err)
	} else {
		inputstr := string(input)
		form_card.National_id = strings.TrimSpace(inputstr)
	}
	_, err = db.Exec(
		"INSERT INTO students (cardUID, created_at, name, national_id, last_in, present) VALUES (?,?,?,?,?,?)",
		cardUID,
		form_card.Created_at,
		form_card.Student_name,
		form_card.National_id,
		form_card.LastIn.Format("2006-01-02 15:04:05"),
		1,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			fmt.Println("The national ID already exists. Please, enter a unique national ID.")
			fmt.Print("\n--------------------------------------------\n\n")
		} else {
			return fmt.Errorf("error inserting INTO students: %s", err.Error())
		}
	}
	fmt.Print("\n--------------------------------------------\n\n")
	return nil
}

func existsDB(Rfid string) (exists bool, err error) {
	var card Rfid_card
	card.Rfid = Rfid
	row := db.QueryRow("SELECT name, national_id, created_at, cardUID, present FROM students WHERE cardUID = ?", card.Rfid)
	switch err := row.Scan(&card.Student_name, &card.National_id, &card.Created_at, &card.Rfid, &card.Is_present); err {
	case sql.ErrNoRows:
		return false, nil
	default:
		return true, nil
	}
}

func updateDB(conn *websocket.Conn) error {
	row := db.QueryRow("SELECT name, national_id, created_at, cardUID, present, last_in, last_out FROM students WHERE cardUID =?", form_card.Rfid)
	if err := row.Scan(
		&form_card.Student_name,
		&form_card.National_id,
		&form_card.Created_at,
		&form_card.Rfid,
		&form_card.Is_present,
		&form_card.LastIn,
		&form_card.LastOut,
	); err != nil {
		return fmt.Errorf("error in updateDB, finding the row: %v", err)
	}

	if form_card.Is_present != 0 {
		form_card.LastOut = time.Now()
		_, err := db.Exec("UPDATE students SET present = 0, last_out = ? WHERE cardUID = ?", form_card.LastOut.Format("2006-01-02 15:04:05"), form_card.Rfid)
		if err != nil {
			return fmt.Errorf("error in updateDB, absent section: %w", err)
		}
		form_card.Is_present = 0
		sendNameToClient(conn, "exit")
	} else {
		form_card.LastIn = time.Now()
		_, err := db.Exec("UPDATE students SET present = 1, last_in = ? WHERE cardUID = ?", form_card.LastIn.Format("2006-01-02 15:04:05"), form_card.Rfid)
		if err != nil {
			return fmt.Errorf("error in updateDB, present section: %w", err)
		}
		form_card.Is_present = 1
		sendNameToClient(conn, "enter")
	}
	return nil
}

func getAllCards() ([]Rfid_card, error) {
	rows, err := db.Query("SELECT name, national_id, created_at, cardUID, last_in, last_out, present FROM students")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []Rfid_card
	for rows.Next() {
		var card Rfid_card
		if err := rows.Scan(&card.Student_name, &card.National_id, &card.Created_at, &card.Rfid, &card.LastIn, &card.LastOut, &card.Is_present); err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	return cards, nil
}

// Handle WebSocket connections and messages
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		err = handleWebSocketMessage(msg, conn)
		if err != nil {
			log.Println("Handle message error:", err)
			break
		}
	}
	http.Redirect(w, r, r.URL.String(), http.StatusFound)
}

func sendNameToClient(conn *websocket.Conn, msgType string) {
	// Query the database for the name associated with the RFID
	if msgType == "register" {
		msg := map[string]string{
			"type": msgType,
			"name": "",
		}
		messageBytes, err := json.Marshal(msg)
		if err != nil {
			log.Println(fmt.Errorf("JSON marshal error: %w", err))
			return
		}
		// Send the message
		err = conn.WriteMessage(websocket.TextMessage, messageBytes)
		if err != nil {
			log.Println(fmt.Errorf("webSocket send error: %w", err))
			return
		}
	}
	var name string
	err := db.QueryRow("SELECT name FROM students WHERE cardUID = ?", form_card.Rfid).Scan(&name)
	if err != nil {
		log.Println(fmt.Errorf("database query error: %w", err))
		return
	}
	// Create a JSON message to send
	msg := map[string]string{
		"type": msgType,
		"name": name,
	}
	messageBytes, err := json.Marshal(msg)
	if err != nil {
		log.Println("JSON marshal error:", err)
		return
	}

	// Send the message
	err = conn.WriteMessage(websocket.TextMessage, messageBytes)
	if err != nil {
		log.Println("WebSocket send error:", err)
		return
	}
}

func handleWebSocketMessage(msg []byte, conn *websocket.Conn) error {
	if err := json.Unmarshal(msg, &form_card); err != nil {
		return fmt.Errorf("JSON unmarshal error: %w", err)
	}

	form_card.Rfid = strings.Replace(form_card.Rfid, " ", "", -1)

	exists, err := existsDB(form_card.Rfid)
	if err != nil {
		return fmt.Errorf("database existence check error: %w", err)
	}

	if !exists {
		sendNameToClient(conn, "register")
		err := newStudent(form_card.Rfid)
		if err != nil {
			return fmt.Errorf("new student creation error: %w", err)
		}
	} else {
		err = updateDB(conn)
		if err != nil {
			return fmt.Errorf("database update error: %w", err)
		}
	}

	return nil
}

func serveIndexTemplate(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("./templates/index.gohtml"))
	cardsData, err := getAllCards()
	if err != nil {
		log.Fatal(err)
	}
	tmpl.Execute(w, cardsData)
}

func togglePresenceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		rfid := r.FormValue("rfid")
		row := db.QueryRow("SELECT present FROM students WHERE cardUID = ?", rfid)
		var isPresent int8
		if err := row.Scan(&isPresent); err != nil {
			http.Error(w, "RFID not found", http.StatusNotFound)
			return
		}

		if isPresent == 1 {
			_, err := db.Exec("UPDATE students SET present = 0, last_out = ? WHERE cardUID = ?", time.Now().Format("2006-01-02 15:04:05"), rfid)
			if err != nil {
				http.Error(w, "Error updating presence", http.StatusInternalServerError)
				return
			}
		} else {
			_, err := db.Exec("UPDATE students SET present = 1, last_in = ? WHERE cardUID = ?", time.Now().Format("2006-01-02 15:04:05"), rfid)
			if err != nil {
				http.Error(w, "Error updating presence", http.StatusInternalServerError)
				return
			}
		}
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func main() {
	// Start the HTTP server
	go func() {
		http.HandleFunc("/", serveIndexTemplate)
		http.HandleFunc("/ws", wsHandler)
		http.HandleFunc("/toggle", togglePresenceHandler)
		log.Fatal(http.ListenAndServe(":8000", nil))
	}()

	select {} // Keep the main goroutine running
}
