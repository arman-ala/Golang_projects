CREATE DATABASE IF NOT EXISTS OnlineClinic;

USE OnlineClinic;

-- Drop existing tables in correct order
DROP TABLE IF EXISTS appointments;
DROP TABLE IF EXISTS prescriptions;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS chats;
DROP TABLE IF EXISTS doctor_availability;
DROP TABLE IF EXISTS patients;
DROP TABLE IF EXISTS doctors;


CREATE TABLE doctors (
    id INT AUTO_INCREMENT PRIMARY KEY,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    national_code CHAR(10) NOT NULL UNIQUE,
    gender ENUM('man', 'woman') NOT NULL,
    phone_number CHAR(11) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    age INT NULL,
    education VARCHAR(100) NULL,
    address TEXT NULL,
    profile_photo_path VARCHAR(255) NULL
) AUTO_INCREMENT = 1;


CREATE TABLE patients (
    id INT AUTO_INCREMENT PRIMARY KEY,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    national_code CHAR(10) NOT NULL UNIQUE,
    gender ENUM('man', 'woman') NOT NULL,
    phone_number CHAR(11) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    age INT,
    job VARCHAR(100),
    education VARCHAR(100),
    address TEXT,
    profile_photo_path VARCHAR(255)
) AUTO_INCREMENT = 1000000;


-- Modified appointments table
CREATE TABLE appointments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    patient_id INT NOT NULL,
    doctor_id INT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    visit_type ENUM('online', 'in-person') NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (doctor_id) REFERENCES doctors(id)
);

-- Modified prescriptions table structure
CREATE TABLE prescriptions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    appointment_id INT NOT NULL,
    instructions TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (appointment_id) REFERENCES appointments(id)
);
ALTER TABLE prescriptions ADD CONSTRAINT unique_appointment UNIQUE (appointment_id);

-- New medications table
CREATE TABLE medications (
    id INT AUTO_INCREMENT PRIMARY KEY,
    prescription_id INT NOT NULL,
    medicine VARCHAR(255) NOT NULL,
    frequency VARCHAR(255) NOT NULL,
    FOREIGN KEY (prescription_id) REFERENCES prescriptions(id)
);

CREATE TABLE IF NOT EXISTS chats (
    id INT AUTO_INCREMENT PRIMARY KEY,
    sender_id INT NOT NULL,
    receiver_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
    id INT AUTO_INCREMENT PRIMARY KEY,
    chat_id INT NOT NULL,
    sender_id INT NOT NULL,
    receiver_id INT NOT NULL, -- Add this column if missing
    text TEXT NOT NULL,
    time VARCHAR(10) NOT NULL,
    replied_message TEXT NULL,
    replied_message_id INT NULL,
    date VARCHAR(10) NOT NULL,
    attached_file_path VARCHAR(255),
    is_read BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (chat_id) REFERENCES chats(id),
    FOREIGN KEY (replied_message_id) REFERENCES messages(id)
);

CREATE TABLE doctor_availability (
    id INT AUTO_INCREMENT PRIMARY KEY,
    doctor_id INT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    type ENUM('online', 'in-person') NOT NULL,
    FOREIGN KEY (doctor_id) REFERENCES doctors(id)
);




DELIMITER //

CREATE PROCEDURE AddDoctor(
    IN p_first_name VARCHAR(50),
    IN p_last_name VARCHAR(50),
    IN p_national_code CHAR(10),
    IN p_gender ENUM('man', 'woman'),
    IN p_phone_number CHAR(11),
    IN p_password VARCHAR(255)
)
BEGIN
    INSERT INTO doctors (
        first_name, last_name, national_code, gender,
        phone_number, password
    ) 
    VALUES (
        p_first_name, p_last_name, p_national_code, p_gender,
        p_phone_number, p_password
    );
END //

-- Modified AddPatient procedure
CREATE PROCEDURE AddPatient(
    IN p_first_name VARCHAR(50),
    IN p_last_name VARCHAR(50),
    IN p_national_code CHAR(10),
    IN p_gender ENUM('man', 'woman'),
    IN p_phone_number CHAR(11),
    IN p_password VARCHAR(255),
    IN p_age INT,
    IN p_job VARCHAR(100),
    IN p_education VARCHAR(100),
    IN p_address TEXT,
    IN p_profile_photo_path VARCHAR(255)
)
BEGIN
    INSERT INTO patients (
        first_name, last_name, national_code, gender, 
        phone_number, password, age,
        job, education, address, profile_photo_path
    ) 
    VALUES (
        p_first_name, p_last_name, p_national_code, p_gender,
        p_phone_number, p_password, p_age,
        p_job, p_education, p_address, p_profile_photo_path
    );
END //


-- Modified UpdatePatient procedure
CREATE PROCEDURE UpdatePatient(
    IN p_id INT,
    IN p_first_name VARCHAR(50),
    IN p_last_name VARCHAR(50),
    IN p_national_code CHAR(10),
    IN p_gender ENUM('man', 'woman'),
    IN p_phone_number CHAR(11),
    IN p_age INT,
    IN p_job VARCHAR(100),
    IN p_education VARCHAR(100),
    IN p_address TEXT,
    IN p_profile_photo_path VARCHAR(255)
)
BEGIN
    UPDATE patients
    SET 
        first_name = p_first_name,
        last_name = p_last_name,
        national_code = p_national_code,
        gender = p_gender,
        phone_number = p_phone_number,
        age = p_age,
        job = p_job,
        education = p_education,
        address = p_address,
        profile_photo_path = p_profile_photo_path
    WHERE id = p_id;
END //

-- Add separate procedure for password updates
CREATE PROCEDURE UpdatePatientPassword(
    IN p_id INT,
    IN p_password VARCHAR(255)
)
BEGIN
    UPDATE patients
    SET password = p_password
    WHERE id = p_id;
END //


-- Modified UpdateDoctor procedure
CREATE PROCEDURE UpdateDoctor(
    IN p_id INT,
    IN p_first_name VARCHAR(50),
    IN p_last_name VARCHAR(50),
    IN p_national_code CHAR(10),
    IN p_gender ENUM('man', 'woman'),
    IN p_phone_number CHAR(11),
    IN p_age INT,
    IN p_education VARCHAR(100),
    IN p_address TEXT,
    IN p_profile_photo_path VARCHAR(255)
)
BEGIN
    UPDATE doctors
    SET 
        first_name = p_first_name,
        last_name = p_last_name,
        national_code = p_national_code,
        gender = p_gender,
        phone_number = p_phone_number,
        age = p_age,
        education = p_education,
        address = p_address,
        profile_photo_path = p_profile_photo_path
    WHERE id = p_id;
END //

-- Create a separate procedure for password updates
CREATE PROCEDURE UpdateDoctorPassword(
    IN p_id INT,
    IN p_password VARCHAR(255)
)
BEGIN
    UPDATE doctors
    SET password = p_password
    WHERE id = p_id;
END //


CREATE PROCEDURE GetPatientByPhone(
    IN p_phone_number CHAR(11)
)
BEGIN
    SELECT 
        id, first_name, last_name, national_code, gender,
        phone_number, password, profile_photo_path
    FROM patients 
    WHERE phone_number = p_phone_number;
END //

CREATE PROCEDURE GetDoctorByPhone(
    IN p_phone_number CHAR(11)
)
BEGIN
    SELECT 
        id, first_name, last_name, national_code, gender,
        phone_number, password, profile_photo_path
    FROM doctors 
    WHERE phone_number = p_phone_number;
END //

CREATE PROCEDURE GetPatientById(
    IN p_id INT
)
BEGIN
    SELECT 
        id, first_name, last_name, national_code, gender,
        phone_number, profile_photo_path
    FROM patients 
    WHERE id = p_id;
END //

CREATE PROCEDURE GetDoctorById(
    IN p_id INT
)
BEGIN
    SELECT 
        id, first_name, last_name, national_code, gender,
        phone_number, profile_photo_path
    FROM doctors 
    WHERE id = p_id;
END //

CREATE PROCEDURE GetAllPatients()
BEGIN
    SELECT 
        id, first_name, last_name, national_code, gender,
        phone_number, profile_photo_path
    FROM patients;
END //

CREATE PROCEDURE GetAllDoctors()
BEGIN
    SELECT 
        id, first_name, last_name, national_code, gender,
        phone_number, profile_photo_path
    FROM doctors;
END //

-- Keep existing procedures that don't need changes
CREATE PROCEDURE AddPrescription(
    IN p_patient_id INT,
    IN p_doctor_id INT,
    IN p_prescription_text TEXT
)
BEGIN
    INSERT INTO prescriptions (patient_id, doctor_id, prescription_text) 
    VALUES (p_patient_id, p_doctor_id, p_prescription_text);
END //

CREATE PROCEDURE GetPatientPrescriptions(
    IN p_id INT
)
BEGIN
    SELECT 
        p.id, p.patient_id, p.doctor_id, p.prescription_text, 
        DATE_FORMAT(p.created_at, '%Y-%m-%d %H:%i:%s') as created_at,
        CONCAT(d.first_name, ' ', d.last_name) as doctor_name
    FROM prescriptions p
    LEFT JOIN doctors d ON p.doctor_id = d.id
    WHERE p.patient_id = p_id;
END //

CREATE PROCEDURE GetDoctorPrescriptions(
    IN d_id INT
)
BEGIN
    SELECT 
        p.id, p.patient_id, p.doctor_id, p.prescription_text, p.created_at,
        CONCAT(pt.first_name, ' ', pt.last_name) as patient_name
    FROM prescriptions p
    JOIN patients pt ON p.patient_id = pt.id
    WHERE p.doctor_id = d_id
    ORDER BY p.created_at DESC;
END //

CREATE PROCEDURE GetChatMessages(
    IN p_doctor_id INT,
    IN p_patient_id INT
)
BEGIN
    SELECT 
        c.id,
        c.sender_id,
        c.receiver_id,
        c.created_at,
        m.id as message_id,
        m.type,
        m.message,
        m.path,
        m.date
    FROM chats c
    LEFT JOIN messages m ON c.id = m.chat_id
    WHERE (
        (c.sender_id = p_doctor_id AND c.receiver_id = p_patient_id) OR
        (c.sender_id = p_patient_id AND c.receiver_id = p_doctor_id)
    )
    ORDER BY c.created_at ASC, m.date ASC;
END //

CREATE PROCEDURE ValidateParticipants(
    IN p_doctor_id INT,
    IN p_patient_id INT,
    OUT is_valid BOOLEAN
)
BEGIN
    DECLARE doctor_exists BOOLEAN;
    DECLARE patient_exists BOOLEAN;
    
    SELECT EXISTS(SELECT 1 FROM doctors WHERE id = p_doctor_id) INTO doctor_exists;
    SELECT EXISTS(SELECT 1 FROM patients WHERE id = p_patient_id) INTO patient_exists;
    
    SET is_valid = doctor_exists AND patient_exists;
END //

DELIMITER ;


-- Procedure to check for overlapping availability slots
DELIMITER //

CREATE PROCEDURE check_availability_overlap(
    IN p_doctor_id INT,
    IN p_start_time DATETIME,
    IN p_end_time DATETIME,
    OUT p_has_overlap BOOLEAN
)
BEGIN
    DECLARE v_count INT;
    
    SELECT COUNT(*) INTO v_count
    FROM doctor_availability
    WHERE doctor_id = p_doctor_id
    AND (
        (start_time <= p_start_time AND end_time > p_start_time)
        OR (start_time < p_end_time AND end_time >= p_end_time)
        OR (start_time >= p_start_time AND end_time <= p_end_time)
    );
    
    SET p_has_overlap = (v_count > 0);
END //

-- Procedure to insert availability slot with overlap check
CREATE PROCEDURE insert_availability_slot(
    IN p_doctor_id INT,
    IN p_start_time DATETIME,
    IN p_end_time DATETIME,
    IN p_type VARCHAR(20),
    OUT p_success BOOLEAN
)
BEGIN
    DECLARE v_has_overlap BOOLEAN;
    DECLARE EXIT HANDLER FOR SQLEXCEPTION
    BEGIN
        SET p_success = FALSE;
        ROLLBACK;
    END;

    START TRANSACTION;
    
    CALL check_availability_overlap(p_doctor_id, p_start_time, p_end_time, v_has_overlap);
    
    IF NOT v_has_overlap THEN
        INSERT INTO doctor_availability (doctor_id, start_time, end_time, type)
        VALUES (p_doctor_id, p_start_time, p_end_time, p_type);
        SET p_success = TRUE;
    ELSE
        SET p_success = FALSE;
    END IF;
    
    COMMIT;
END //

-- Procedure to get available slots for a doctor within a date range
CREATE PROCEDURE get_available_slots(
    IN p_doctor_id INT,
    IN p_start_date DATE,
    IN p_end_date DATE
)
BEGIN
    SELECT id, doctor_id, start_time, end_time, type
    FROM doctor_availability
    WHERE doctor_id = p_doctor_id
    AND DATE(start_time) >= p_start_date
    AND DATE(end_time) <= p_end_date
    ORDER BY start_time;
END //

DELIMITER ;


DELIMITER //

DROP PROCEDURE IF EXISTS get_available_slots //

CREATE PROCEDURE get_available_slots(
    IN p_doctor_id INT,
    IN p_start_date DATETIME,
    IN p_end_date DATETIME
)
BEGIN
    SELECT 
        id,
        doctor_id,
        -- Ensure consistent datetime format
        DATE_FORMAT(start_time, '%Y-%m-%d %H:%i:%s') as start_time,
        DATE_FORMAT(end_time, '%Y-%m-%d %H:%i:%s') as end_time,
        type
    FROM doctor_availability
    WHERE doctor_id = p_doctor_id
        AND start_time >= p_start_date
        AND end_time <= p_end_date
        AND start_time > NOW()  -- Only show future slots
    ORDER BY start_time ASC;
END //

DELIMITER ;


DELIMITER //

CREATE PROCEDURE delete_unreserved_availability(
    IN p_doctor_id INT,
    IN p_visit_type VARCHAR(20),
    OUT p_deleted_count INT
)
BEGIN
    -- Get count before deletion
    SELECT COUNT(*) INTO p_deleted_count
    FROM doctor_availability
    WHERE doctor_id = p_doctor_id
        AND type = p_visit_type
        AND start_time > NOW();

    -- Delete unreserved slots
    DELETE FROM doctor_availability
    WHERE doctor_id = p_doctor_id
        AND type = p_visit_type
        AND start_time > NOW();
END //

DELIMITER ;


DELIMITER //

-- Procedure to get prescription by appointment with access control

CREATE PROCEDURE GetPrescriptionByAppointmentSecure(
    IN p_appointment_id INT,
    IN p_user_id INT,
    IN p_is_doctor BOOLEAN
)
BEGIN
    SELECT 
        p.id,
        a.doctor_id,
        a.patient_id,
        p.appointment_id,
        a.visit_type,
        DATE_FORMAT(p.created_at, '%Y-%m-%d') as created_at,
        p.instructions,
        CASE 
            WHEN p_is_doctor THEN CONCAT(pt.first_name, ' ', pt.last_name)
            ELSE CONCAT(d.first_name, ' ', d.last_name)
        END as name,
        m.medicine,
        m.frequency
    FROM prescriptions p
    JOIN appointments a ON p.appointment_id = a.id
    JOIN doctors d ON a.doctor_id = d.id
    JOIN patients pt ON a.patient_id = pt.id
    LEFT JOIN medications m ON p.id = m.prescription_id
    WHERE p.appointment_id = p_appointment_id
    AND (
        (p_is_doctor AND a.doctor_id = p_user_id) OR
        (NOT p_is_doctor AND a.patient_id = p_user_id)
    );
END //


-- Procedure to get patient prescriptions with access control
CREATE PROCEDURE GetPatientPrescriptionsSecure(
    IN p_patient_id INT,
    IN p_user_id INT,
    IN p_is_doctor BOOLEAN
)
BEGIN
    SELECT 
        p.id,
        a.doctor_id,
        a.patient_id,
        p.appointment_id,
        a.visit_type,
        DATE_FORMAT(p.created_at, '%Y-%m-%d') as created_at,
        p.instructions,
        CASE 
            WHEN p_is_doctor THEN CONCAT(pt.first_name, ' ', pt.last_name)
            ELSE CONCAT(d.first_name, ' ', d.last_name)
        END as name
    FROM prescriptions p
    JOIN appointments a ON p.appointment_id = a.id
    JOIN doctors d ON a.doctor_id = d.id
    JOIN patients pt ON a.patient_id = pt.id
    WHERE a.patient_id = p_patient_id
    AND (
        (p_is_doctor AND a.doctor_id = p_user_id) OR
        (NOT p_is_doctor AND a.patient_id = p_user_id)
    );
END //

DELIMITER ;

DELIMITER //

CREATE PROCEDURE CreateChat(
    IN p_sender_id INT,
    IN p_receiver_id INT,
    OUT p_chat_id INT
)
BEGIN
    DECLARE existing_chat_id INT;

    -- Check if a chat already exists between the two participants
    SELECT id INTO existing_chat_id
    FROM chats
    WHERE (sender_id = p_sender_id AND receiver_id = p_receiver_id)
       OR (sender_id = p_receiver_id AND receiver_id = p_sender_id)
    LIMIT 1;

    -- If a chat already exists, return the existing chat ID
    IF existing_chat_id IS NOT NULL THEN
        SET p_chat_id = existing_chat_id;
    ELSE
        -- If no chat exists, create a new one
        INSERT INTO chats (sender_id, receiver_id) VALUES (p_sender_id, p_receiver_id);
        SET p_chat_id = LAST_INSERT_ID();
    END IF;

    -- Return the chat ID
    SELECT p_chat_id AS chat_id;
END //

DELIMITER ;


DELIMITER //

CREATE PROCEDURE AddMessage(
    IN p_chat_id INT,
    IN p_sender_id INT,
    IN p_receiver_id INT,
    IN p_text TEXT,
    IN p_time VARCHAR(10),
    IN p_replied_message TEXT,
    IN p_replied_message_id INT,
    IN p_date VARCHAR(10),
    IN p_attached_file_path VARCHAR(255),
    IN p_is_read BOOLEAN
)
BEGIN
    INSERT INTO messages (
        chat_id,
        sender_id,
        receiver_id,
        text,
        time,
        replied_message,
        replied_message_id,
        date,
        attached_file_path,
        is_read
    )
    VALUES (
        p_chat_id,
        p_sender_id,
        p_receiver_id,
        p_text,
        p_time,
        NULLIF(p_replied_message, ''), -- Set to NULL if empty string
        NULLIF(p_replied_message_id, 0), -- Set to NULL if 0
        p_date,
        p_attached_file_path,
        p_is_read
    );
END //

DELIMITER ;


DELIMITER //

CREATE PROCEDURE GetChatHistory(
    IN p_user_id INT,
    IN p_receiver_id INT
)
BEGIN
    -- Retrieve chat history
    SELECT 
        c.id AS chat_id,
        c.sender_id,
        c.receiver_id,
        DATE_FORMAT(c.created_at, '%Y-%m-%d %H:%i:%s') AS chat_created_at,
        COALESCE(m.id, 0) AS message_id,
        COALESCE(m.text, '') AS text,
        COALESCE(m.time, '') AS time,
        COALESCE(m.replied_message, '') AS replied_message,
        COALESCE(m.replied_message_id, 0) AS replied_message_id,
        COALESCE(m.date, '') AS date,
        COALESCE(m.sender_id, 0) AS message_sender_id,
        COALESCE(m.receiver_id, 0) AS message_receiver_id,
        COALESCE(m.attached_file_path, '') AS attached_file_path,
        COALESCE(m.is_read, FALSE) AS is_read
    FROM chats c
    LEFT JOIN messages m ON c.id = m.chat_id
    WHERE (c.sender_id = p_user_id AND c.receiver_id = p_receiver_id)
       OR (c.sender_id = p_receiver_id AND c.receiver_id = p_user_id)
    ORDER BY c.created_at ASC, m.id ASC; -- Order by chat creation time and message ID

    -- Mark messages as read
    UPDATE messages m
    JOIN chats c ON m.chat_id = c.id
    SET m.is_read = TRUE
    WHERE (
        (c.sender_id = p_user_id AND c.receiver_id = p_receiver_id)
        OR (c.sender_id = p_receiver_id AND c.receiver_id = p_user_id)
    )
    AND m.sender_id != p_user_id
    AND m.is_read = FALSE;
END //

DELIMITER ;

DELIMITER //

-- Function to convert Gregorian date to Solar date
CREATE FUNCTION GregorianToSolar(gregorian_date DATE) RETURNS DATE
DETERMINISTIC
BEGIN
    DECLARE solar_date DATE;
    -- Implement the conversion logic from Gregorian to Solar (Hijri) date
    -- This is a placeholder, you need to implement the actual conversion logic
    SET solar_date = DATE_SUB(gregorian_date, INTERVAL 579 DAY); -- Example conversion
    RETURN solar_date;
END //

-- Function to get the current Solar year
CREATE FUNCTION SolarYear() RETURNS INT
DETERMINISTIC
BEGIN
    DECLARE solar_year INT;
    SET solar_year = YEAR(GregorianToSolar(CURDATE()));
    RETURN solar_year;
END //

-- Function to get the current Solar date
CREATE FUNCTION SolarDate() RETURNS DATE
DETERMINISTIC
BEGIN
    RETURN GregorianToSolar(CURDATE());
END //

DELIMITER ;


DELIMITER //

CREATE FUNCTION GetCurrentSolarDate() RETURNS VARCHAR(10)
DETERMINISTIC
BEGIN
    DECLARE solarDate VARCHAR(10);
    SET solarDate = DATE_FORMAT(NOW(), '%Y-%m-%d'); -- Replace with Solar date conversion logic
    RETURN solarDate;
END //

DELIMITER ;

DELIMITER //

CREATE PROCEDURE GetAllChats(
    IN p_user_id INT
)
BEGIN
    SELECT 
        c.id AS chat_id,
        CASE 
            WHEN c.sender_id = p_user_id THEN c.receiver_id
            ELSE c.sender_id
        END AS other_user_id,
        CASE 
            WHEN c.sender_id = p_user_id THEN (
                SELECT CONCAT(first_name, ' ', last_name) 
                FROM patients 
                WHERE id = c.receiver_id
                UNION ALL
                SELECT CONCAT(first_name, ' ', last_name) 
                FROM doctors 
                WHERE id = c.receiver_id
                LIMIT 1
            )
            ELSE (
                SELECT CONCAT(first_name, ' ', last_name) 
                FROM patients 
                WHERE id = c.sender_id
                UNION ALL
                SELECT CONCAT(first_name, ' ', last_name) 
                FROM doctors 
                WHERE id = c.sender_id
                LIMIT 1
            )
        END AS other_user_name,
        CASE 
            WHEN c.sender_id = p_user_id THEN (
                SELECT profile_photo_path 
                FROM patients 
                WHERE id = c.receiver_id
                UNION ALL
                SELECT profile_photo_path 
                FROM doctors 
                WHERE id = c.receiver_id
                LIMIT 1
            )
            ELSE (
                SELECT profile_photo_path 
                FROM patients 
                WHERE id = c.sender_id
                UNION ALL
                SELECT profile_photo_path 
                FROM doctors 
                WHERE id = c.sender_id
                LIMIT 1
            )
        END AS other_user_image
    FROM chats c
    WHERE c.sender_id = p_user_id OR c.receiver_id = p_user_id;
END //

DELIMITER ;