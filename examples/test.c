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
} Product;

void Product_init_table() {
    char *sql = "CREATE TABLE IF NOT EXISTS products ("
        "id INTEGER"
        ","
        "name TEXT"
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

Product* Product_create(int64_t id, char* name) {
    char sql[1024];
    sprintf(sql, "INSERT INTO products (id, name) VALUES (?, ?)");

    sqlite3_stmt *stmt;
    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to prepare statement: %s\n", sqlite3_errmsg(db));
        return NULL;
    }

    sqlite3_bind_int64(stmt, 1, id);
    sqlite3_bind_text(stmt, 2, name, -1, SQLITE_TRANSIENT);

    rc = sqlite3_step(stmt);
    if (rc != SQLITE_DONE) {
        fprintf(stderr, "Failed to insert: %s\n", sqlite3_errmsg(db));
        sqlite3_finalize(stmt);
        return NULL;
    }

    int64_t last_insert_id = sqlite3_last_insert_rowid(db);
    sqlite3_finalize(stmt);

    Product* obj = (Product*)malloc(sizeof(Product));
    obj->id = id;
    obj->name = strdup(name);

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

int main(int argc, char *argv[]) {
    // Initialize database
    int rc = sqlite3_open("app.db", &db);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Cannot open database: %s\n", sqlite3_errmsg(db));
        return 1;
    }
    printf("Database opened successfully\n\n");

    Product_init_table();

    // ===== CRUD DEMO =====
    printf("\n===== CRUD Operations Demo =====\n\n");

    // CREATE: Insert records
    printf("Creating records...\n");
    Product* product1 = Product_create(10, "Class A");
    if (product1) printf("  Created Product with ID: %lld\n", product1->id);

    Product* product2 = Product_create(20, "Class B");
    if (product2) printf("  Created Product with ID: %lld\n", product2->id);

    Product* product3 = Product_create(30, "Class A");
    if (product3) printf("  Created Product with ID: %lld\n", product3->id);

    // READ: Find by ID
    printf("\nFinding record by ID...\n");
    Product* found = Product_find(1);
    if (found) {
        printf("  Found Product ID %lld: ", found->id);
        printf("id=%lld ", found->id);
        printf("name=%s ", found->name);
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
