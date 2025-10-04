#include <Wire.h>
#include <SPI.h>
#include <MFRC522.h>
#include <WiFi.h>
#include <ArduinoJson.h>
#include <LiquidCrystal_I2C.h>
#include <WebSocketsClient.h>

#define RST_PIN 27
#define SS_PIN 5
#define BUZZER_PIN 13

MFRC522 mfrc522(SS_PIN, RST_PIN);  // Create MFRC522 instance
WiFiClient espClient;
LiquidCrystal_I2C lcd(0x27, 16, 2); // Adjust the address if necessary
WebSocketsClient webSocket;

long lastMsg = 0;
char msg[50];
int value = 0;

char ssid[] = "HUAWEI P30 Pro"; // your network SSID (name)
char pass[] = "12345678"; // your network password

void webSocketEvent(WStype_t type, uint8_t * payload, size_t length) {
  switch(type) {
    case WStype_DISCONNECTED:
        Serial.println("WebSocket Disconnected!"); // Log disconnection message to the serial monitor
        break;
    case WStype_CONNECTED:
        Serial.println("WebSocket Connected!"); // Log connection message to the serial monitor
        break;
    case WStype_TEXT:
        {
        Serial.printf("WebSocket Message: %s\n", payload); // Print received WebSocket text message to serial monitor
        StaticJsonDocument<200> doc; // Create a JSON document with enough capacity
        DeserializationError error = deserializeJson(doc, payload); // Parse JSON from the payload
        if (error) {
            Serial.print("deserializeJson() failed: "); // Log error message if JSON deserialization fails
            Serial.println(error.f_str());
            return; // Exit the function on error
        }
        const char* messageType = doc["type"]; // Extract "type" field from JSON document
        const char* name = doc["name"]; // Extract "name" field from JSON document
        lcd.clear(); // Clear the LCD display
        lcd.setCursor(0, 0); // Set LCD cursor to first line, first column
        if (strcmp(messageType, "enter") == 0) { // Check if messageType is "enter"
            lcd.print("Welcome to class"); // Display welcome message on LCD
        } else if(strcmp(messageType, "register") == 0) { // Check if messageType is "register"
            lcd.print("Please Register!"); // Display registration prompt on LCD
        } else {
            lcd.print("Goodbye!"); // Display goodbye message on LCD
        }
        lcd.setCursor(0, 1); // Set LCD cursor to second line, first column
        lcd.print(name); // Display the name on the second line of LCD
        break; // Exit switch-case block
    }
  }
}


void setup() {
  pinMode(BUZZER_PIN, OUTPUT);
  Serial.begin(115200); // Initialize serial communications with the PC
  SPI.begin();          // Init SPI bus
  mfrc522.PCD_Init();   // Init MFRC522 
  WiFi.begin(ssid, pass);

  // Initialize the I2C communication with default SDA and SCL pins
  Wire.begin(4, 22); // SDA on GPIO4, SCL on GPIO22

  // Initialize the LCD
  lcd.begin(16, 2); // Initialize the lcd with 16 columns and 2 rows
  lcd.backlight();  // Turn on the backlight

  // Print a welcome message to the LCD
  lcd.setCursor(0, 0); // Set the cursor to the first column and first row
  lcd.print("Initializing...");
  lcd.setCursor(0, 1); // Set the cursor to the first column and second row
  lcd.print("Please wait...");

  // Connect to WiFi
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
    lcd.setCursor(0, 1); // Update the message on the second row
    lcd.print("Connecting...");
  }

  Serial.println("\nConnected to WiFi");
  lcd.setCursor(0, 0); // Clear the first row
  lcd.print("WiFi Connected");
  delay(2000); // Wait for 2 seconds
  lcd.clear(); // Clear the LCD

  // Initialize WebSocket
  webSocket.begin("192.168.43.233", 8000, "/ws");  // Set your server IP address and port
  webSocket.onEvent(webSocketEvent);  // Set WebSocket event handler

  webSocket.setReconnectInterval(5000);  // Set WebSocket to reconnect every 5 seconds if the connection is lost
}

void loop() {
  webSocket.loop();  

  if (mfrc522.PICC_IsNewCardPresent()) {
    if (mfrc522.PICC_ReadCardSerial()) {
      String content = "";
      for (byte i = 0; i < mfrc522.uid.size; i++) {
        content.concat(String(mfrc522.uid.uidByte[i] < 0x10 ? " 0" : " "));
        content.concat(String(mfrc522.uid.uidByte[i], HEX));
      }
      content.toUpperCase();
      while (WiFi.status() != WL_CONNECTED) {
      delay(500);
      Serial.print(".");
      lcd.setCursor(0, 1); 
      lcd.print("Connecting...");
    }
      Serial.println("\nConnected to WiFi");
      lcd.setCursor(0, 0); 
      delay(2000); 
      lcd.clear(); 
      lcd.setCursor(0, 1);
      lcd.clear();

      String jsonMessage = prepareJSON(content.c_str());
      
      webSocket.sendTXT(jsonMessage);

      Serial.println("Message sent: " + jsonMessage);

      // Buzz the buzzer
      digitalWrite(BUZZER_PIN, HIGH); 
      delay(700);
      digitalWrite(BUZZER_PIN, LOW);
    }
  }
}

String prepareJSON(const char* id) {
  StaticJsonDocument<200> doc;
  doc["RFID"] = id;
  String jsonString;
  serializeJson(doc, jsonString);
  return jsonString;
}
