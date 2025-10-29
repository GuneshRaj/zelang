#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sqlite3.h>

// Global database connection
sqlite3 *db = NULL;

// Struct: Product
typedef struct Product {
    int64_t id;
    char* name;
    double price;
    int64_t quantity;
} Product;

// Struct: Category
typedef struct Category {
    int64_t id;
    char* name;
} Category;

void Product_init_table() {
    char *sql = "CREATE TABLE IF NOT EXISTS products ("
        "id INTEGER PRIMARY KEY AUTOINCREMENT"
        ","
        "name TEXT NOT NULL"
        ","
        "price REAL NOT NULL"
        ","
        "quantity INTEGER"
        ")";
    
    char *err_msg = NULL;
    int rc = sqlite3_exec(db, sql, NULL, NULL, &err_msg);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "SQL error: %s\n", err_msg);
        sqlite3_free(err_msg);
    } else {
        printf("Table products created successfully\n");
    }
}

Product* Product_create(char* name, double price, int64_t quantity) {
    char sql[1024];
    sprintf(sql, "INSERT INTO products (name, price, quantity) VALUES (?, ?, ?)");

    sqlite3_stmt *stmt;
    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to prepare statement: %s\n", sqlite3_errmsg(db));
        return NULL;
    }

    sqlite3_bind_text(stmt, 1, name, -1, SQLITE_TRANSIENT);
    sqlite3_bind_double(stmt, 2, price);
    sqlite3_bind_int64(stmt, 3, quantity);

    rc = sqlite3_step(stmt);
    if (rc != SQLITE_DONE) {
        fprintf(stderr, "Failed to insert: %s\n", sqlite3_errmsg(db));
        sqlite3_finalize(stmt);
        return NULL;
    }

    int64_t last_insert_id = sqlite3_last_insert_rowid(db);
    sqlite3_finalize(stmt);

    Product* obj = (Product*)malloc(sizeof(Product));
    obj->id = last_insert_id;
    obj->name = strdup(name);
    obj->price = price;
    obj->quantity = quantity;

    return obj;
}

Product* Product_find(int64_t id) {
    char *sql = "SELECT * FROM products WHERE id = ?";
    sqlite3_stmt *stmt;

    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to prepare statement: %s\n", sqlite3_errmsg(db));
        return NULL;
    }

    sqlite3_bind_int64(stmt, 1, id);

    rc = sqlite3_step(stmt);
    if (rc != SQLITE_ROW) {
        sqlite3_finalize(stmt);
        return NULL;
    }

    Product* obj = (Product*)malloc(sizeof(Product));
    obj->id = sqlite3_column_int64(stmt, 0);
    obj->name = strdup((const char*)sqlite3_column_text(stmt, 1));
    obj->price = sqlite3_column_double(stmt, 2);
    obj->quantity = sqlite3_column_int64(stmt, 3);

    sqlite3_finalize(stmt);
    return obj;
}

Product** Product_all(int* count) {
    char *sql = "SELECT * FROM products";
    sqlite3_stmt *stmt;

    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to prepare statement: %s\n", sqlite3_errmsg(db));
        *count = 0;
        return NULL;
    }

    int capacity = 10;
    Product** results = (Product**)malloc(capacity * sizeof(Product*));
    int n = 0;

    while ((rc = sqlite3_step(stmt)) == SQLITE_ROW) {
        if (n >= capacity) {
            capacity *= 2;
            results = (Product**)realloc(results, capacity * sizeof(Product*));
        }

        Product* obj = (Product*)malloc(sizeof(Product));
        obj->id = sqlite3_column_int64(stmt, 0);
        obj->name = strdup((const char*)sqlite3_column_text(stmt, 1));
        obj->price = sqlite3_column_double(stmt, 2);
        obj->quantity = sqlite3_column_int64(stmt, 3);

        results[n++] = obj;
    }

    sqlite3_finalize(stmt);
    *count = n;
    return results;
}

int Product_delete(int64_t id) {
    char *sql = "DELETE FROM products WHERE id = ?";
    sqlite3_stmt *stmt;

    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to prepare statement: %s\n", sqlite3_errmsg(db));
        return 0;
    }

    sqlite3_bind_int64(stmt, 1, id);

    rc = sqlite3_step(stmt);
    sqlite3_finalize(stmt);

    if (rc != SQLITE_DONE) {
        fprintf(stderr, "Failed to delete: %s\n", sqlite3_errmsg(db));
        return 0;
    }

    return 1;
}

void Category_init_table() {
    char *sql = "CREATE TABLE IF NOT EXISTS categories ("
        "id INTEGER PRIMARY KEY AUTOINCREMENT"
        ","
        "name TEXT NOT NULL"
        ")";
    
    char *err_msg = NULL;
    int rc = sqlite3_exec(db, sql, NULL, NULL, &err_msg);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "SQL error: %s\n", err_msg);
        sqlite3_free(err_msg);
    } else {
        printf("Table categories created successfully\n");
    }
}

Category* Category_create(char* name) {
    char sql[1024];
    sprintf(sql, "INSERT INTO categories (name) VALUES (?)");

    sqlite3_stmt *stmt;
    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to prepare statement: %s\n", sqlite3_errmsg(db));
        return NULL;
    }

    sqlite3_bind_text(stmt, 1, name, -1, SQLITE_TRANSIENT);

    rc = sqlite3_step(stmt);
    if (rc != SQLITE_DONE) {
        fprintf(stderr, "Failed to insert: %s\n", sqlite3_errmsg(db));
        sqlite3_finalize(stmt);
        return NULL;
    }

    int64_t last_insert_id = sqlite3_last_insert_rowid(db);
    sqlite3_finalize(stmt);

    Category* obj = (Category*)malloc(sizeof(Category));
    obj->id = last_insert_id;
    obj->name = strdup(name);

    return obj;
}

Category* Category_find(int64_t id) {
    char *sql = "SELECT * FROM categories WHERE id = ?";
    sqlite3_stmt *stmt;

    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to prepare statement: %s\n", sqlite3_errmsg(db));
        return NULL;
    }

    sqlite3_bind_int64(stmt, 1, id);

    rc = sqlite3_step(stmt);
    if (rc != SQLITE_ROW) {
        sqlite3_finalize(stmt);
        return NULL;
    }

    Category* obj = (Category*)malloc(sizeof(Category));
    obj->id = sqlite3_column_int64(stmt, 0);
    obj->name = strdup((const char*)sqlite3_column_text(stmt, 1));

    sqlite3_finalize(stmt);
    return obj;
}

Category** Category_all(int* count) {
    char *sql = "SELECT * FROM categories";
    sqlite3_stmt *stmt;

    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to prepare statement: %s\n", sqlite3_errmsg(db));
        *count = 0;
        return NULL;
    }

    int capacity = 10;
    Category** results = (Category**)malloc(capacity * sizeof(Category*));
    int n = 0;

    while ((rc = sqlite3_step(stmt)) == SQLITE_ROW) {
        if (n >= capacity) {
            capacity *= 2;
            results = (Category**)realloc(results, capacity * sizeof(Category*));
        }

        Category* obj = (Category*)malloc(sizeof(Category));
        obj->id = sqlite3_column_int64(stmt, 0);
        obj->name = strdup((const char*)sqlite3_column_text(stmt, 1));

        results[n++] = obj;
    }

    sqlite3_finalize(stmt);
    *count = n;
    return results;
}

int Category_delete(int64_t id) {
    char *sql = "DELETE FROM categories WHERE id = ?";
    sqlite3_stmt *stmt;

    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to prepare statement: %s\n", sqlite3_errmsg(db));
        return 0;
    }

    sqlite3_bind_int64(stmt, 1, id);

    rc = sqlite3_step(stmt);
    sqlite3_finalize(stmt);

    if (rc != SQLITE_DONE) {
        fprintf(stderr, "Failed to delete: %s\n", sqlite3_errmsg(db));
        return 0;
    }

    return 1;
}

int main(int argc, char *argv[]) {
    // Initialize database
    int rc = sqlite3_open("app.db", &db);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Cannot open database: %s\n", sqlite3_errmsg(db));
        return 1;
    }
    printf("Database opened successfully\n\n");

    Product_init_table();
    Category_init_table();

    // ===== CRUD DEMO =====
    printf("\n===== CRUD Operations Demo =====\n\n");

    // CREATE: Insert records
    printf("Creating records...\n");
    Product* product1 = Product_create("John Doe", 10, 10);
    if (product1) printf("  Created Product with ID: %lld\n", product1->id);

    Product* product2 = Product_create("Jane Smith", 20, 20);
    if (product2) printf("  Created Product with ID: %lld\n", product2->id);

    Product* product3 = Product_create("Bob Johnson", 30, 30);
    if (product3) printf("  Created Product with ID: %lld\n", product3->id);

    // READ: Find by ID
    printf("\nFinding record by ID...\n");
    Product* found = Product_find(1);
    if (found) {
        printf("  Found Product ID %lld: ", found->id);
        printf("id=%lld ", found->id);
        printf("name=%s ", found->name);
        printf("price=%f ", found->price);
        printf("quantity=%lld ", found->quantity);
        printf("\n");
    }

    // READ: Get all records
    printf("\nGetting all records...\n");
    int count = 0;
    Product** all = Product_all(&count);
    printf("  Found %d records:\n", count);
    for (int i = 0; i < count; i++) {
        printf("    [%d] ID=%lld", i+1, all[i]->id);
        printf(" name=%s", all[i]->name);
        printf("\n");
    }

    // DELETE: Remove a record
    printf("\nDeleting record with ID=2...\n");
    int deleted = Product_delete(2);
    if (deleted) printf("  Record deleted successfully\n");

    // Verify deletion
    printf("\nVerifying deletion...\n");
    all = Product_all(&count);
    printf("  Remaining records: %d\n", count);

    printf("\n===== Demo Complete =====\n");

    // Close database
    sqlite3_close(db);
    return 0;
}
