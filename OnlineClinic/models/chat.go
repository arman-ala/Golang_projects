// ./models/chat.go
package models

import (
	"database/sql"
	"fmt"
	"onlineClinic/utils"
	"strings"
	"time"
)

type Chat struct {
	ID         int       `json:"id"`
	SenderID   int       `json:"sender_id"`
	ReceiverID int       `json:"receiver_id"`
	CreatedAt  time.Time `json:"created_at"`
	Messages   []Message `json:"messages"`
}

type Message struct {
	ID               int64          `json:"id"`
	ChatID           int64          `json:"chat_id"`
	Text             string         `json:"text"`
	Time             string         `json:"time"`
	RepliedMessage   *string        `json:"replied_message,omitempty"`
	RepliedMessageID *int64         `json:"replied_message_id,omitempty"`
	Date             sql.NullString `json:"date"` // Use sql.NullString for handling NULL values
	SenderID         int64          `json:"sender_id"`
	ReceiverID       int64          `json:"receiver_id"`
	AttachedFile     *File          `json:"attached_file,omitempty"`
	IsRead           bool           `json:"is_read"`
}

type MessageResponse struct {
	ID               int64   `json:"id"`
	ChatID           int64   `json:"chat_id"`
	Text             string  `json:"text"`
	Time             string  `json:"time"`
	RepliedMessage   *string `json:"replied_message,omitempty"`
	RepliedMessageID *int64  `json:"replied_message_id,omitempty"`
	Date             string  `json:"date,omitempty"` // Use omitempty to exclude if empty
	SenderID         int64   `json:"sender_id"`
	ReceiverID       int64   `json:"receiver_id"`
	AttachedFile     *File   `json:"attached_file,omitempty"`
	IsRead           bool    `json:"is_read"`
}

type File struct {
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type             string  `json:"type"`               // Message type (e.g., "chat")
	ChatID           int     `json:"chat_id"`            // ID of the chat
	SenderID         int     `json:"sender_id"`          // ID of the sender
	ReceiverID       int     `json:"receiver_id"`        // ID of the receiver
	Message          string  `json:"message"`            // Text message (optional)
	Time             string  `json:"time"`               // Time of the message
	RepliedMessage   *string `json:"replied_message"`    // Replied message text (nullable)
	RepliedMessageID *int    `json:"replied_message_id"` // ID of the replied message (nullable)
	Date             string  `json:"date"`               // Date of the message
	AttachedFile     *File   `json:"attached_file"`      // Attached file details (nullable)
	IsRead           bool    `json:"is_read"`            // Whether the message has been read
}

func CreateChat(db *sql.DB, senderID, receiverID int) (int, error) {
	// Call the stored procedure to create the chat
	_, err := db.Exec("CALL CreateChat(?, ?, @chat_id)", senderID, receiverID)
	if err != nil {
		return 0, err
	}

	// Retrieve the chat_id using a separate query
	var chatID int
	err = db.QueryRow("SELECT @chat_id").Scan(&chatID)
	if err != nil {
		return 0, err
	}

	return chatID, nil
}

func AddMessage(db *sql.DB, chatID, senderID, receiverID int, text, timeStr, repliedMessage string, repliedMessageID *int, hijriDate string, attachedFile *File, isRead bool) error {
	// Convert Hijri date to Gregorian date if necessary
	gregorianDate, err := utils.SolarToGregorian(hijriDate)
	if err != nil {
		return fmt.Errorf("error converting Hijri date to Gregorian: %v", err)
	}

	// Format the Gregorian date as a string for the database
	date := gregorianDate.Format("2006-01-02")

	existingChatID, err := ChatExists(db, senderID, receiverID)
	if err != nil {
		return fmt.Errorf("error checking for existing chat: %v", err)
	}

	// log.Printf("Existing Chat ID: %d, Provided Chat ID: %d", existingChatID, chatID)

	if existingChatID == 0 {
		return fmt.Errorf("no chat exists between sender %d and receiver %d", senderID, receiverID)
	}

	if chatID != 0 && chatID != existingChatID {
		return fmt.Errorf("invalid chat ID: provided chat ID %d does not match existing chat ID %d", chatID, existingChatID)
	}

	var filePath sql.NullString
	if attachedFile != nil {
		filePath.String = attachedFile.URL
		filePath.Valid = true
	}

	_, err = db.Exec(
		"CALL AddMessage(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		existingChatID, // Use the existing chat ID
		senderID,
		receiverID,
		text,
		timeStr, // Use the provided time string
		repliedMessage,
		repliedMessageID,
		date, // Use the Gregorian date here
		filePath,
		isRead,
	)
	return err
}

func GetChatHistory(db *sql.DB, userID, receiverID int) ([]Chat, error) {
	rows, err := db.Query("CALL GetChatHistory(?, ?)", userID, receiverID)
	if err != nil {
		// log.Printf("Error calling GetChatHistory stored procedure: %v", err)
		return nil, err
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var chat Chat
		var message Message
		var repliedMessageDB sql.NullString
		var repliedMessageIDDB sql.NullInt64
		var filePath sql.NullString
		var chatCreatedAtStr string
		var messageDateDB sql.NullString // Use sql.NullString for the date field

		err := rows.Scan(
			&chat.ID,
			&chat.SenderID,
			&chat.ReceiverID,
			&chatCreatedAtStr,
			&message.ID,
			&message.Text,
			&message.Time,
			&repliedMessageDB,
			&repliedMessageIDDB,
			&messageDateDB, // Scan into sql.NullString
			&message.SenderID,
			&message.ReceiverID,
			&filePath,
			&message.IsRead,
		)
		if err != nil {
			// log.Printf("Error scanning row: %v", err)
			return nil, err
		}

		// Parse chat_created_at into a time.Time
		chat.CreatedAt, err = time.Parse("2006-01-02 15:04:05", chatCreatedAtStr)
		if err != nil {
			// log.Printf("Error parsing chat_created_at: %v", err)
			return nil, err
		}

		// Handle NULL values for replied_message
		if repliedMessageDB.Valid {
			message.RepliedMessage = &repliedMessageDB.String
		} else {
			message.RepliedMessage = nil
		}

		// Handle NULL values for replied_message_id
		if repliedMessageIDDB.Valid {
			message.RepliedMessageID = &repliedMessageIDDB.Int64
		} else {
			message.RepliedMessageID = nil
		}

		// Handle NULL values for the date field
		if messageDateDB.Valid {
			// Parse the Gregorian date from the database
			gregorianDate, err := time.Parse("2006-01-02", messageDateDB.String)
			if err != nil {
				// log.Printf("Error parsing Gregorian date: %v", err)
				return nil, err
			}

			// Convert the Gregorian date to Solar (Hijri) date
			solarDate := utils.GregorianToSolar(gregorianDate)
			message.Date = sql.NullString{String: solarDate, Valid: true}
		} else {
			message.Date = sql.NullString{String: "", Valid: false} // Set to empty if NULL
		}

		// Handle file URL
		if filePath.Valid {
			// Extract file name and type from the URL
			fileName, fileType := extractFileNameAndType(filePath.String)
			message.AttachedFile = &File{
				Name: fileName,
				Type: fileType,
				URL:  filePath.String,
			}
		} else {
			message.AttachedFile = nil
		}

		// Add the message to the chat
		chat.Messages = append(chat.Messages, message)
		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		// log.Printf("Error iterating rows: %v", err)
		return nil, err
	}

	return chats, nil
}

// Helper function to extract file name and type from URL
func extractFileNameAndType(url string) (string, string) {
	// Split the URL by "/" to get the file path
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return "", ""
	}

	// Get the last part of the URL (the file name with extension)
	fileNameWithExt := parts[len(parts)-1]

	// Split the file name and extension
	fileParts := strings.Split(fileNameWithExt, ".")
	if len(fileParts) < 2 {
		return fileNameWithExt, "" // No extension found
	}

	// File name is everything before the last "."
	fileName := strings.Join(fileParts[:len(fileParts)-1], ".")
	// File type is the last part after the last "."
	fileType := fileParts[len(fileParts)-1]

	return fileName, fileType
}

func convertMessageToResponse(message Message) MessageResponse {
	return MessageResponse{
		ID:               message.ID,
		ChatID:           message.ChatID,
		Text:             message.Text,
		Time:             message.Time,
		RepliedMessage:   message.RepliedMessage,
		RepliedMessageID: message.RepliedMessageID,
		Date:             message.Date.String, // Use .String to get the value
		SenderID:         message.SenderID,
		ReceiverID:       message.ReceiverID,
		AttachedFile:     message.AttachedFile,
		IsRead:           message.IsRead,
	}
}

func ChatExists(db *sql.DB, senderID, receiverID int) (int, error) {
	var chatID int
	err := db.QueryRow(`
        SELECT id FROM chats 
        WHERE (sender_id = ? AND receiver_id = ?) 
           OR (sender_id = ? AND receiver_id = ?)`,
		senderID, receiverID, receiverID, senderID,
	).Scan(&chatID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // No chat exists
		}
		return 0, err
	}
	return chatID, nil
}

// GetAllChats retrieves all chats for a user (doctor or patient)
func GetAllChats(db *sql.DB, userID int) ([]map[string]interface{}, error) {
	rows, err := db.Query("CALL GetAllChats(?)", userID)
	if err != nil {
		// log.Printf("Error calling GetAllChats stored procedure: %v", err)
		return nil, err
	}
	defer rows.Close()

	var chats []map[string]interface{}

	for rows.Next() {
		var chatID, otherUserID int
		var otherUserName, otherUserImage sql.NullString

		err := rows.Scan(&chatID, &otherUserID, &otherUserName, &otherUserImage)
		if err != nil {
			// log.Printf("Error scanning row: %v", err)
			return nil, err
		}

		chat := map[string]interface{}{
			"chat_id": chatID,
			"id":      otherUserID,
			"name":    otherUserName.String,  // Use .String to handle NULL values
			"image":   otherUserImage.String, // Use .String to handle NULL values
		}

		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		// log.Printf("Error iterating rows: %v", err)
		return nil, err
	}

	return chats, nil
}

func GetUnreadChats(db *sql.DB, userID int) ([]map[string]interface{}, error) {
	query := `
        SELECT 
            c.id AS chat_id,
            CASE 
                WHEN c.sender_id = ? THEN c.receiver_id
                ELSE c.sender_id
            END AS other_user_id,
            COALESCE(
                (SELECT CONCAT(p.first_name, ' ', p.last_name) FROM patients p WHERE p.id = CASE WHEN c.sender_id = ? THEN c.receiver_id ELSE c.sender_id END),
                (SELECT CONCAT(d.first_name, ' ', d.last_name) FROM doctors d WHERE d.id = CASE WHEN c.sender_id = ? THEN c.receiver_id ELSE c.sender_id END),
                'Unknown'
            ) AS full_name,
            COALESCE(
                (SELECT p.profile_photo_path FROM patients p WHERE p.id = CASE WHEN c.sender_id = ? THEN c.receiver_id ELSE c.sender_id END),
                (SELECT d.profile_photo_path FROM doctors d WHERE d.id = CASE WHEN c.sender_id = ? THEN c.receiver_id ELSE c.sender_id END),
                ''
            ) AS profile_photo_path
        FROM chats c
        JOIN messages m ON c.id = m.chat_id
        WHERE (c.sender_id = ? OR c.receiver_id = ?)
        AND m.sender_id != ?
        AND m.is_read = FALSE
        GROUP BY c.id
    `

	rows, err := db.Query(query, userID, userID, userID, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var unreadChats []map[string]interface{}
	for rows.Next() {
		var chatID, otherUserID int
		var fullName, profilePhotoPath string

		err := rows.Scan(&chatID, &otherUserID, &fullName, &profilePhotoPath)
		if err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}

		unreadChats = append(unreadChats, map[string]interface{}{
			"chat_id":            chatID,
			"other_user_id":      otherUserID,
			"full_name":          fullName,
			"profile_photo_path": profilePhotoPath,
		})
	}

	return unreadChats, nil
}
