-- Staff rows reference hospitals, so remove them first (created via /staff/create during demos).
DELETE FROM patients WHERE hospital_id IN (SELECT id FROM hospitals WHERE code IN ('hospital-a', 'hospital-b'));
DELETE FROM staff    WHERE hospital_id IN (SELECT id FROM hospitals WHERE code IN ('hospital-a', 'hospital-b'));
DELETE FROM hospitals WHERE code IN ('hospital-a', 'hospital-b');
