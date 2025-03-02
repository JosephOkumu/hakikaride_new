CREATE TABLE IF NOT EXISTS Students (
    StudentID INTEGER PRIMARY KEY AUTOINCREMENT,
    ParentID INTEGER,
    FirstName TEXT NOT NULL,
    LastName TEXT NOT NULL,
    Grade TEXT NOT NULL,
    AdmNumber TEXT NOT NULL,
    Address TEXT NOT NULL,
    EmergencyContact TEXT NOT NULL,
    IsActive BOOLEAN DEFAULT true,
    FOREIGN KEY (ParentID) REFERENCES Parents(ParentID)
);
