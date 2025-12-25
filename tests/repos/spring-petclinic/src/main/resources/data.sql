-- Pet Types
INSERT INTO pet_types (id, name) VALUES (1, 'cat');
INSERT INTO pet_types (id, name) VALUES (2, 'dog');
INSERT INTO pet_types (id, name) VALUES (3, 'lizard');
INSERT INTO pet_types (id, name) VALUES (4, 'snake');
INSERT INTO pet_types (id, name) VALUES (5, 'bird');
INSERT INTO pet_types (id, name) VALUES (6, 'hamster');

-- Specialties
INSERT INTO specialties (id, name) VALUES (1, 'radiology');
INSERT INTO specialties (id, name) VALUES (2, 'surgery');
INSERT INTO specialties (id, name) VALUES (3, 'dentistry');

-- Vets
INSERT INTO vets (id, first_name, last_name) VALUES (1, 'James', 'Carter');
INSERT INTO vets (id, first_name, last_name) VALUES (2, 'Helen', 'Leary');
INSERT INTO vets (id, first_name, last_name) VALUES (3, 'Linda', 'Douglas');
INSERT INTO vets (id, first_name, last_name) VALUES (4, 'Rafael', 'Ortega');
INSERT INTO vets (id, first_name, last_name) VALUES (5, 'Henry', 'Stevens');
INSERT INTO vets (id, first_name, last_name) VALUES (6, 'Sharon', 'Jenkins');

-- Vet Specialties
INSERT INTO vet_specialties (vet_id, specialty_id) VALUES (2, 1);
INSERT INTO vet_specialties (vet_id, specialty_id) VALUES (3, 2);
INSERT INTO vet_specialties (vet_id, specialty_id) VALUES (3, 3);
INSERT INTO vet_specialties (vet_id, specialty_id) VALUES (4, 2);
INSERT INTO vet_specialties (vet_id, specialty_id) VALUES (5, 1);

-- Owners
INSERT INTO owners (id, first_name, last_name, address, city, telephone, email) VALUES (1, 'George', 'Franklin', '110 W. Liberty St.', 'Madison', '6085551023', 'george@example.com');
INSERT INTO owners (id, first_name, last_name, address, city, telephone, email) VALUES (2, 'Betty', 'Davis', '638 Cardinal Ave.', 'Sun Prairie', '6085551749', 'betty@example.com');
INSERT INTO owners (id, first_name, last_name, address, city, telephone, email) VALUES (3, 'Eduardo', 'Rodriquez', '2693 Commerce St.', 'McFarland', '6085558763', 'eduardo@example.com');
INSERT INTO owners (id, first_name, last_name, address, city, telephone, email) VALUES (4, 'Harold', 'Davis', '563 Friendly St.', 'Windsor', '6085553198', 'harold@example.com');
INSERT INTO owners (id, first_name, last_name, address, city, telephone, email) VALUES (5, 'Peter', 'McTavish', '2387 S. Fair Way', 'Madison', '6085552765', 'peter@example.com');

-- Pets
INSERT INTO pets (id, name, birth_date, type_id, owner_id) VALUES (1, 'Leo', '2020-09-07', 1, 1);
INSERT INTO pets (id, name, birth_date, type_id, owner_id) VALUES (2, 'Basil', '2012-08-06', 6, 2);
INSERT INTO pets (id, name, birth_date, type_id, owner_id) VALUES (3, 'Rosy', '2011-04-17', 2, 3);
INSERT INTO pets (id, name, birth_date, type_id, owner_id) VALUES (4, 'Jewel', '2010-03-07', 2, 3);
INSERT INTO pets (id, name, birth_date, type_id, owner_id) VALUES (5, 'Iggy', '2010-11-30', 3, 4);
INSERT INTO pets (id, name, birth_date, type_id, owner_id) VALUES (6, 'George', '2010-01-20', 4, 5);

-- Visits
INSERT INTO visits (id, pet_id, vet_id, visit_date, description) VALUES (1, 1, 1, CURRENT_DATE, 'Annual checkup');
INSERT INTO visits (id, pet_id, vet_id, visit_date, description) VALUES (2, 2, 2, CURRENT_DATE, 'Dental cleaning');
INSERT INTO visits (id, pet_id, vet_id, visit_date, description) VALUES (3, 3, 3, '2024-01-15', 'Surgery follow-up');
