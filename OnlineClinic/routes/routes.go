package routes

import (
	"net/http"
	"onlineClinic/controllers"
	"onlineClinic/utils"
	"os"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

var Hub *controllers.Hub

func SetupRoutes(router *mux.Router) http.Handler {
	// Public routes
	router.HandleFunc("/api/login/patient", controllers.LoginPatient).Methods("POST")
	router.HandleFunc("/api/login/doctor", controllers.LoginDoctor).Methods("POST")
	router.HandleFunc("/api/register/patient", controllers.RegisterPatient).Methods("POST")
	router.HandleFunc("/api/register/doctor", controllers.RegisterDoctor).Methods("POST")
	if os.Getenv("ENV") != "production" {
		router.HandleFunc("/api/debug/verify-hash", controllers.VerifyStoredHash).Methods("GET")
	}

	// Protected routes
	api := router.PathPrefix("/api").Subrouter()
	api.Use(utils.AuthMiddleware)

	// Doctor routes
	api.HandleFunc("/allDoctors/search", controllers.SearchDoctors).Methods("POST")
	api.HandleFunc("/doctors/{id}", utils.DoctorAuthMiddleware(controllers.GetDoctorProfile)).Methods("GET")
	api.HandleFunc("/doctors/{id}", utils.DoctorAuthMiddleware(controllers.UpdateDoctorProfile)).Methods("PUT")
	api.HandleFunc("/doctors/{id}", utils.DoctorAuthMiddleware(controllers.DeleteDoctorProfile)).Methods("DELETE")
	api.HandleFunc("/doc/password", utils.DoctorAuthMiddleware(controllers.UpdateDoctorPassword)).Methods("PUT")
	api.HandleFunc("/doctors", controllers.GetAllDoctors).Methods("GET")
	api.HandleFunc("/doctors/{id}/2nearestAppointments", utils.DoctorAuthMiddleware(controllers.GetDoctorTwoNearestAppointments)).Methods("GET")
	api.HandleFunc("/doctors/{id}/photo", utils.DoctorAuthMiddleware(controllers.DeleteDoctorProfilePhoto)).Methods("DELETE")

	// Doctor Availability Management
	api.HandleFunc("/doctors/{id}/availability", utils.DoctorOrPatientAuthMiddleware(controllers.GetDoctorAvailability)).Methods("GET")
	api.HandleFunc("/doctors/{id}/availability", utils.DoctorAuthMiddleware(controllers.SetDoctorAvailability)).Methods("POST")
	api.HandleFunc("/doctors/{id}/availability/{slotId}", utils.DoctorAuthMiddleware(controllers.DeleteDoctorAvailability)).Methods("DELETE")

	// Patient routes
	api.HandleFunc("/pnt/password", utils.PatientAuthMiddleware(controllers.UpdatePatientPassword)).Methods("PUT")
	api.HandleFunc("/patients/{id}", utils.PatientAuthMiddleware(controllers.GetPatientProfile)).Methods("GET")
	api.HandleFunc("/patient/{id}", utils.DoctorOrPatientAuthMiddleware(controllers.GetPatientInfo)).Methods("GET")
	api.HandleFunc("/patients/{id}", utils.PatientAuthMiddleware(controllers.UpdatePatientProfile)).Methods("PUT")
	api.HandleFunc("/patients/{id}", utils.PatientAuthMiddleware(controllers.DeletePatientProfile)).Methods("DELETE")
	api.HandleFunc("/patients/{id}/photo", utils.PatientAuthMiddleware(controllers.DeletePatientProfilePhoto)).Methods("DELETE")

	// Appointment routes
	api.HandleFunc("/appointments", utils.PatientAuthMiddleware(controllers.CreateAppointment)).Methods("POST")
	api.HandleFunc("/patients/{id}/2nearestAppointments", utils.PatientAuthMiddleware(controllers.GetPatientTwoNearestAppointments)).Methods("GET")
	api.HandleFunc("/patients/{id}/appointments", utils.DoctorOrPatientAuthMiddleware(controllers.GetPatientAppointments)).Methods("GET")
	api.HandleFunc("/appointments/{id}", utils.DoctorOrPatientAuthMiddleware(controllers.DeleteAppointment)).Methods("DELETE")
	api.HandleFunc("/doctors/{id}/appointments", utils.DoctorAuthMiddleware(controllers.GetDoctorAppointments)).Methods("GET")
	api.HandleFunc("/patients/{id}/all_appointments", utils.DoctorOrPatientAuthMiddleware(controllers.GetPatientAllAppointments)).Methods("GET")
	api.HandleFunc("/doctors/{id}/all_appointments", utils.DoctorAuthMiddleware(controllers.GetDoctorAllAppointments)).Methods("GET")
	api.HandleFunc("/doctor/unreservedAvailableTimes", utils.DoctorAuthMiddleware(controllers.DeleteUnreservedAvailability)).Methods("DELETE")
	//api.HandleFunc("/prescriptions", utils.DoctorAuthMiddleware(controllers.CreatePrescription)).Methods("POST")
	api.HandleFunc("/prescriptions/search", utils.DoctorOrPatientAuthMiddleware(controllers.GetPrescriptionsByPatientNameAndDateHandler)).Methods("GET")
	api.HandleFunc("/prescriptions/patient/{id}", utils.DoctorOrPatientAuthMiddleware(controllers.GetPrescriptionsByPatient)).Methods("GET")
	api.HandleFunc("/prescriptions/doctor/{id}", utils.DoctorAuthMiddleware(controllers.GetPrescriptionsByDoctor)).Methods("GET")
	api.HandleFunc("/prescriptions/{appointmentId}", utils.DoctorOrPatientAuthMiddleware(controllers.GetPrescriptionByAppointment)).Methods("GET")
	api.HandleFunc("/prescriptions", utils.DoctorAuthMiddleware(controllers.UpdatePrescription)).Methods("PUT")

	// File upload routes
	api.HandleFunc("/upload/profile", controllers.UploadProfilePhoto).Methods("POST")

	// Chat routes
	api.HandleFunc("/chat", utils.DoctorOrPatientAuthMiddleware(controllers.CreateChat)).Methods("POST")
	api.HandleFunc("/chat/history", utils.DoctorOrPatientAuthMiddleware(controllers.GetChatHistory)).Methods("GET")
	api.HandleFunc("/chats/unread", utils.DoctorOrPatientAuthMiddleware(controllers.GetUnreadChats)).Methods("GET")

	// File upload route
	api.HandleFunc("/chats", utils.DoctorOrPatientAuthMiddleware(controllers.GetAllChats)).Methods("GET")
	api.HandleFunc("/upload/chat", controllers.UploadChatFile).Methods("POST")

	// WebSocket route (under router, not api)
	Hub = controllers.NewHub()
	go Hub.Run()
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		controllers.ServeWs(Hub, w, r)
	})

	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://kashan-clininc.liara.run", "https://online-clinic.liara.run", "http://185.142.159.118"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Origin", "Accept"},
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: true,
		Debug:            true,
	})

	return c.Handler(router)
}
