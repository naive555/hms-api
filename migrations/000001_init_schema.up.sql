CREATE TABLE hospitals (
  id            BIGSERIAL PRIMARY KEY,
  code          VARCHAR(50)  NOT NULL UNIQUE,
  name          VARCHAR(255) NOT NULL,
  his_base_url  VARCHAR(255) NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE staff (
    id            BIGSERIAL PRIMARY KEY,
    hospital_id   BIGINT       NOT NULL REFERENCES hospitals(id),
    username      VARCHAR(50)  NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_staff_hospital_username UNIQUE (hospital_id, username)
);

CREATE TABLE patients (
    id             BIGSERIAL PRIMARY KEY,
    hospital_id    BIGINT      NOT NULL REFERENCES hospitals(id),
    patient_hn     VARCHAR(50) NOT NULL,
    first_name_th  VARCHAR(100),
    middle_name_th VARCHAR(100),
    last_name_th   VARCHAR(100),
    first_name_en  VARCHAR(100),
    middle_name_en VARCHAR(100),
    last_name_en   VARCHAR(100),
    date_of_birth  DATE,
    national_id    VARCHAR(13),
    passport_id    VARCHAR(20),
    phone_number   VARCHAR(20),
    email          VARCHAR(255),
    gender         CHAR(1),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_patients_hospital_hn UNIQUE (hospital_id, patient_hn),
    CONSTRAINT ck_patients_gender CHECK (gender IN ('M', 'F')),
    CONSTRAINT ck_patients_has_id CHECK (national_id IS NOT NULL OR passport_id IS NOT NULL)
);

CREATE UNIQUE INDEX uq_patients_hospital_national_id
    ON patients (hospital_id, national_id) WHERE national_id IS NOT NULL;
CREATE UNIQUE INDEX uq_patients_hospital_passport_id
    ON patients (hospital_id, passport_id) WHERE passport_id IS NOT NULL;
-- Search indexes. Every query is scoped by hospital_id, so it leads each index.
-- Name/email use lower(...) text_pattern_ops to support case-insensitive prefix search:
--   lower(first_name_en) LIKE lower($1) || '%'
CREATE INDEX ix_patients_first_name_th  ON patients (hospital_id, lower(first_name_th)  text_pattern_ops);
CREATE INDEX ix_patients_first_name_en  ON patients (hospital_id, lower(first_name_en)  text_pattern_ops);
CREATE INDEX ix_patients_middle_name_th ON patients (hospital_id, lower(middle_name_th) text_pattern_ops);
CREATE INDEX ix_patients_middle_name_en ON patients (hospital_id, lower(middle_name_en) text_pattern_ops);
CREATE INDEX ix_patients_last_name_th   ON patients (hospital_id, lower(last_name_th)   text_pattern_ops);
CREATE INDEX ix_patients_last_name_en   ON patients (hospital_id, lower(last_name_en)   text_pattern_ops);
CREATE INDEX ix_patients_date_of_birth  ON patients (hospital_id, date_of_birth);
CREATE INDEX ix_patients_phone_number   ON patients (hospital_id, phone_number);
CREATE INDEX ix_patients_email          ON patients (hospital_id, lower(email));
