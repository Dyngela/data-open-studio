-- PostgreSQL Test Data
-- Sample data for testing

-- Insert employees
INSERT INTO employees (first_name, last_name, email, department, salary, hire_date, is_active) VALUES
('Jean', 'Dupont', 'jean.dupont@company.com', 'Engineering', 75000.00, '2020-03-15', true),
('Marie', 'Martin', 'marie.martin@company.com', 'Marketing', 65000.00, '2019-07-22', true),
('Pierre', 'Bernard', 'pierre.bernard@company.com', 'Engineering', 82000.00, '2018-01-10', true),
('Sophie', 'Petit', 'sophie.petit@company.com', 'HR', 58000.00, '2021-05-03', true),
('Lucas', 'Robert', 'lucas.robert@company.com', 'Sales', 70000.00, '2020-11-18', true),
('Emma', 'Richard', 'emma.richard@company.com', 'Engineering', 78000.00, '2019-09-01', true),
('Hugo', 'Durand', 'hugo.durand@company.com', 'Finance', 72000.00, '2017-04-25', true),
('Lea', 'Leroy', 'lea.leroy@company.com', 'Marketing', 61000.00, '2022-02-14', true),
('Thomas', 'Moreau', 'thomas.moreau@company.com', 'Engineering', 85000.00, '2016-08-30', true),
('Camille', 'Simon', 'camille.simon@company.com', 'Sales', 68000.00, '2021-01-07', false);

-- Insert products
INSERT INTO products (name, category, price, stock_quantity, supplier) VALUES
('Laptop Pro 15', 'Electronics', 1299.99, 50, 'TechSupply Inc'),
('Wireless Mouse', 'Electronics', 29.99, 200, 'TechSupply Inc'),
('Office Chair Ergo', 'Furniture', 349.99, 30, 'OfficePlus'),
('Standing Desk', 'Furniture', 599.99, 15, 'OfficePlus'),
('USB-C Hub', 'Electronics', 49.99, 100, 'TechSupply Inc'),
('Monitor 27"', 'Electronics', 399.99, 40, 'DisplayWorld'),
('Keyboard Mechanical', 'Electronics', 129.99, 75, 'TechSupply Inc'),
('Webcam HD', 'Electronics', 79.99, 60, 'TechSupply Inc'),
('Desk Lamp LED', 'Furniture', 45.99, 90, 'LightCo'),
('Notebook Pack', 'Office Supplies', 12.99, 500, 'PaperWorld');

-- Insert customers
INSERT INTO customers (first_name, last_name, email, phone, city, country) VALUES
('Alice', 'Johnson', 'alice.johnson@email.com', '+1-555-0101', 'New York', 'USA'),
('Bob', 'Smith', 'bob.smith@email.com', '+1-555-0102', 'Los Angeles', 'USA'),
('Claire', 'Dubois', 'claire.dubois@email.com', '+33-1-23456789', 'Paris', 'France'),
('David', 'Mueller', 'david.mueller@email.com', '+49-30-12345678', 'Berlin', 'Germany'),
('Elena', 'Garcia', 'elena.garcia@email.com', '+34-91-1234567', 'Madrid', 'Spain'),
('Frank', 'Wilson', 'frank.wilson@email.com', '+44-20-12345678', 'London', 'UK'),
('Giulia', 'Rossi', 'giulia.rossi@email.com', '+39-02-12345678', 'Milan', 'Italy'),
('Hans', 'Schmidt', 'hans.schmidt@email.com', '+49-89-12345678', 'Munich', 'Germany'),
('Isabelle', 'Lefevre', 'isabelle.lefevre@email.com', '+33-4-12345678', 'Lyon', 'France'),
('James', 'Brown', 'james.brown@email.com', '+1-555-0110', 'Chicago', 'USA');

-- Insert orders
INSERT INTO orders (customer_id, order_date, status, total_amount, shipping_address) VALUES
(1, '2024-01-15 10:30:00', 'completed', 1329.98, '123 Main St, New York, NY 10001'),
(2, '2024-01-16 14:22:00', 'completed', 449.98, '456 Oak Ave, Los Angeles, CA 90001'),
(3, '2024-01-17 09:15:00', 'shipped', 599.99, '12 Rue de la Paix, 75002 Paris'),
(4, '2024-01-18 16:45:00', 'processing', 179.98, 'Berliner Str 45, 10115 Berlin'),
(5, '2024-01-19 11:00:00', 'pending', 1699.98, 'Calle Mayor 78, 28013 Madrid'),
(1, '2024-01-20 13:30:00', 'completed', 79.99, '123 Main St, New York, NY 10001'),
(6, '2024-01-21 15:20:00', 'shipped', 929.98, '10 Baker Street, London W1U 3BW'),
(7, '2024-01-22 10:00:00', 'processing', 349.99, 'Via Roma 25, 20121 Milano'),
(8, '2024-01-23 12:45:00', 'pending', 259.98, 'Maximilianstr 10, 80539 Munich'),
(3, '2024-01-24 09:30:00', 'completed', 529.98, '12 Rue de la Paix, 75002 Paris');

-- Insert order items
INSERT INTO order_items (order_id, product_id, quantity, unit_price) VALUES
(1, 1, 1, 1299.99),
(1, 2, 1, 29.99),
(2, 3, 1, 349.99),
(2, 5, 2, 49.99),
(3, 4, 1, 599.99),
(4, 7, 1, 129.99),
(4, 5, 1, 49.99),
(5, 1, 1, 1299.99),
(5, 6, 1, 399.99),
(6, 8, 1, 79.99),
(7, 6, 2, 399.99),
(7, 7, 1, 129.99),
(8, 3, 1, 349.99),
(9, 2, 2, 29.99),
(9, 9, 2, 45.99),
(9, 10, 10, 12.99),
(10, 5, 2, 49.99),
(10, 7, 1, 129.99),
(10, 6, 1, 399.99);
