#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sqlite3.h>

// Global database connection
sqlite3 *db = NULL;

// Struct: Student
typedef struct Student {
    int64_t id;
    char* name;
    char* class;
} Student;

void Student_init_table() {
    char *sql = "CREATE TABLE IF NOT EXISTS students ("
        "id INTEGER"
        ","
        "name TEXT"
        ","
        "class TEXT"
        ")";
    
    char *err_msg = NULL;
    int rc = sqlite3_exec(db, sql, NULL, NULL, &err_msg);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "SQL error: %s\n", err_msg);
        sqlite3_free(err_msg);
    } else {
        printf("Table students created successfully\n");
    }
}

Student* Student_create(int64_t id, char* name, char* class) {
    char sql[1024];
    sprintf(sql, "INSERT INTO students (id, name, class) VALUES (?, ?, ?)");

    sqlite3_stmt *stmt;
    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to prepare statement: %s\n", sqlite3_errmsg(db));
        return NULL;
    }

    sqlite3_bind_int64(stmt, 1, id);
    sqlite3_bind_text(stmt, 2, name, -1, SQLITE_TRANSIENT);
    sqlite3_bind_text(stmt, 3, class, -1, SQLITE_TRANSIENT);

    rc = sqlite3_step(stmt);
    if (rc != SQLITE_DONE) {
        fprintf(stderr, "Failed to insert: %s\n", sqlite3_errmsg(db));
        sqlite3_finalize(stmt);
        return NULL;
    }

    int64_t last_insert_id = sqlite3_last_insert_rowid(db);
    sqlite3_finalize(stmt);

    Student* obj = (Student*)malloc(sizeof(Student));
    obj->id = id;
    obj->name = strdup(name);
    obj->class = strdup(class);

    return obj;
}

Student* Student_find(int64_t id) {
    char *sql = "SELECT * FROM students WHERE id = ?";
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

    Student* obj = (Student*)malloc(sizeof(Student));
    obj->id = sqlite3_column_int64(stmt, 0);
    obj->name = strdup((const char*)sqlite3_column_text(stmt, 1));
    obj->class = strdup((const char*)sqlite3_column_text(stmt, 2));

    sqlite3_finalize(stmt);
    return obj;
}

Student** Student_all(int* count) {
    char *sql = "SELECT * FROM students";
    sqlite3_stmt *stmt;

    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to prepare statement: %s\n", sqlite3_errmsg(db));
        *count = 0;
        return NULL;
    }

    int capacity = 10;
    Student** results = (Student**)malloc(capacity * sizeof(Student*));
    int n = 0;

    while ((rc = sqlite3_step(stmt)) == SQLITE_ROW) {
        if (n >= capacity) {
            capacity *= 2;
            results = (Student**)realloc(results, capacity * sizeof(Student*));
        }

        Student* obj = (Student*)malloc(sizeof(Student));
        obj->id = sqlite3_column_int64(stmt, 0);
        obj->name = strdup((const char*)sqlite3_column_text(stmt, 1));
        obj->class = strdup((const char*)sqlite3_column_text(stmt, 2));

        results[n++] = obj;
    }

    sqlite3_finalize(stmt);
    *count = n;
    return results;
}

int Student_delete(int64_t id) {
    char *sql = "DELETE FROM students WHERE id = ?";
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

    Student_init_table();

    // ===== CRUD DEMO =====
    printf("\n===== CRUD Operations Demo =====\n\n");

    // CREATE: Insert records
    printf("Creating records...\n");
    Student* student1 = Student_create(10, "Class A", "Sample");
    if (student1) printf("  Created Student with ID: %lld\n", student1->id);

    Student* student2 = Student_create(20, "Class B", "Sample");
    if (student2) printf("  Created Student with ID: %lld\n", student2->id);

    Student* student3 = Student_create(30, "Class A", "Sample");
    if (student3) printf("  Created Student with ID: %lld\n", student3->id);

    // READ: Find by ID
    printf("\nFinding record by ID...\n");
    Student* found = Student_find(1);
    if (found) {
        printf("  Found Student ID %lld: ", found->id);
        printf("id=%lld ", found->id);
        printf("name=%s ", found->name);
        printf("class=%s ", found->class);
        printf("\n");
    }

    // READ: Get all records
    printf("\nGetting all records...\n");
    int count = 0;
    Student** all = Student_all(&count);
    printf("  Found %d records:\n", count);
    for (int i = 0; i < count; i++) {
        printf("    [%d] ID=%lld", i+1, all[i]->id);
        printf(" name=%s", all[i]->name);
        printf("\n");
    }

    // DELETE: Remove a record
    printf("\nDeleting record with ID=2...\n");
    int deleted = Student_delete(2);
    if (deleted) printf("  Record deleted successfully\n");

    // Verify deletion
    printf("\nVerifying deletion...\n");
    all = Student_all(&count);
    printf("  Remaining records: %d\n", count);

    printf("\n===== Demo Complete =====\n");

    // Close database
    sqlite3_close(db);
    return 0;
}
