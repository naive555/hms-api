-- Demo data only. All names, IDs, phone numbers and emails are fictional.
--
-- Scenarios covered:
--   * Hospital isolation: "Somchai" exists in both hospitals; each staff member sees only their own.
--   * Same person in two hospitals: national_id 1234567890123 has a separate record in A and B.
--   * patient_hn is unique per hospital only: HN0001 exists in both A and B.
--   * Foreigners with passport_id only (no national_id).
--   * Prefix search: "Som" matches Somchai, Somying, Somsak in hospital A.
--   * Middle names (TH and EN) and nullable optional fields.

INSERT INTO hospitals (code, name, his_base_url) VALUES
    ('hospital-a', 'Hospital A', 'http://mockhis:8081'),
    ('hospital-b', 'Hospital B', 'https://hospital-b.api.co.th');

-- Hospital A
INSERT INTO patients (
    hospital_id, patient_hn,
    first_name_th, middle_name_th, last_name_th,
    first_name_en, middle_name_en, last_name_en,
    date_of_birth, national_id, passport_id, phone_number, email, gender
) VALUES
    ((SELECT id FROM hospitals WHERE code = 'hospital-a'), 'HN0001',
     'สมชาย', NULL, 'ใจดี',
     'Somchai', NULL, 'Jaidee',
     '1990-05-12', '1234567890123', NULL, '0812345678', 'somchai@example.com', 'M'),

    ((SELECT id FROM hospitals WHERE code = 'hospital-a'), 'HN0002',
     'สมหญิง', NULL, 'รักดี',
     'Somying', NULL, 'Rakdee',
     '1985-11-03', '1100700012345', NULL, '0898765432', 'somying@example.com', 'F'),

    ((SELECT id FROM hospitals WHERE code = 'hospital-a'), 'HN0003',
     'จอห์น', 'ไมเคิล', 'สมิธ',
     'John', 'Michael', 'Smith',
     '1978-02-20', NULL, 'AB1234567', '0923456789', 'john.smith@example.com', 'M'),

    ((SELECT id FROM hospitals WHERE code = 'hospital-a'), 'HN0004',
     'สมศักดิ์', NULL, 'วงศ์สวัสดิ์',
     'Somsak', NULL, 'Wongsawat',
     '2001-07-30', '1509900054321', NULL, '0861112222', NULL, 'M');

-- Hospital B
INSERT INTO patients (
    hospital_id, patient_hn,
    first_name_th, middle_name_th, last_name_th,
    first_name_en, middle_name_en, last_name_en,
    date_of_birth, national_id, passport_id, phone_number, email, gender
) VALUES
    ((SELECT id FROM hospitals WHERE code = 'hospital-b'), 'HN0001',
     'สมชาย', NULL, 'ศรีสุข',
     'Somchai', NULL, 'Srisuk',
     '1992-01-15', '3100600098765', NULL, '0834445555', 'somchai.s@example.com', 'M'),

    ((SELECT id FROM hospitals WHERE code = 'hospital-b'), 'HN0002',
     'สมชาย', NULL, 'ใจดี',
     'Somchai', NULL, 'Jaidee',
     '1990-05-12', '1234567890123', NULL, '0812345678', 'somchai@example.com', 'M'),

    ((SELECT id FROM hospitals WHERE code = 'hospital-b'), 'HN0003',
     'ยูกิ', NULL, 'ทานากะ',
     'Yuki', NULL, 'Tanaka',
     '1995-09-08', NULL, 'TZ7654321', '0957778888', 'yuki.tanaka@example.com', 'F');
