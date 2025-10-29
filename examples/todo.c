#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sqlite3.h>
#include <ctype.h>
#include <microhttpd.h>

// Global database connection
sqlite3 *db = NULL;

// Global HTTP server
struct MHD_Daemon *http_daemon = NULL;

// Struct: Todo
typedef struct Todo {
    int64_t id;
    char* title;
    char* description;
    int completed;
} Todo;

void Todo_init_table() {
    char *sql = "CREATE TABLE IF NOT EXISTS todos ("
        "id INTEGER PRIMARY KEY AUTOINCREMENT"
        ","
        "title TEXT NOT NULL"
        ","
        "description TEXT NOT NULL"
        ","
        "completed INTEGER"
        ")";
    
    char *err_msg = NULL;
    int rc = sqlite3_exec(db, sql, NULL, NULL, &err_msg);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "SQL error: %s\n", err_msg);
        sqlite3_free(err_msg);
    } else {
        printf("Table todos created successfully\n");
    }
}

Todo* Todo_create(char* title, char* description, int completed) {
    char sql[1024];
    sprintf(sql, "INSERT INTO todos (title, description, completed) VALUES (?, ?, ?)");

    sqlite3_stmt *stmt;
    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to prepare statement: %s\n", sqlite3_errmsg(db));
        return NULL;
    }

    sqlite3_bind_text(stmt, 1, title, -1, SQLITE_TRANSIENT);
    sqlite3_bind_text(stmt, 2, description, -1, SQLITE_TRANSIENT);

    rc = sqlite3_step(stmt);
    if (rc != SQLITE_DONE) {
        fprintf(stderr, "Failed to insert: %s\n", sqlite3_errmsg(db));
        sqlite3_finalize(stmt);
        return NULL;
    }

    int64_t last_insert_id = sqlite3_last_insert_rowid(db);
    sqlite3_finalize(stmt);

    Todo* obj = (Todo*)malloc(sizeof(Todo));
    obj->id = last_insert_id;
    obj->title = strdup(title);
    obj->description = strdup(description);
    obj->completed = completed;

    return obj;
}

Todo* Todo_find(int64_t id) {
    char *sql = "SELECT * FROM todos WHERE id = ?";
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

    Todo* obj = (Todo*)malloc(sizeof(Todo));
    obj->id = sqlite3_column_int64(stmt, 0);
    obj->title = strdup((const char*)sqlite3_column_text(stmt, 1));
    obj->description = strdup((const char*)sqlite3_column_text(stmt, 2));

    sqlite3_finalize(stmt);
    return obj;
}

Todo** Todo_all(int* count) {
    char *sql = "SELECT * FROM todos";
    sqlite3_stmt *stmt;

    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to prepare statement: %s\n", sqlite3_errmsg(db));
        *count = 0;
        return NULL;
    }

    int capacity = 10;
    Todo** results = (Todo**)malloc(capacity * sizeof(Todo*));
    int n = 0;

    while ((rc = sqlite3_step(stmt)) == SQLITE_ROW) {
        if (n >= capacity) {
            capacity *= 2;
            results = (Todo**)realloc(results, capacity * sizeof(Todo*));
        }

        Todo* obj = (Todo*)malloc(sizeof(Todo));
        obj->id = sqlite3_column_int64(stmt, 0);
        obj->title = strdup((const char*)sqlite3_column_text(stmt, 1));
        obj->description = strdup((const char*)sqlite3_column_text(stmt, 2));

        results[n++] = obj;
    }

    sqlite3_finalize(stmt);
    *count = n;
    return results;
}

int Todo_delete(int64_t id) {
    char *sql = "DELETE FROM todos WHERE id = ?";
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


// HTML generation functions
const char* html_header =
    "<!DOCTYPE html>\n"
    "<html lang='en'>\n"
    "<head>\n"
    "    <meta charset='UTF-8'>\n"
    "    <meta name='viewport' content='width=device-width, initial-scale=1.0'>\n"
    "    <title>%s</title>\n"
    "    <link href='https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css' rel='stylesheet'>\n"
    "</head>\n"
    "<body>\n"
    "    <div class='container mt-5'>\n";

const char* html_footer =
    "    </div>\n"
    "    <script src='https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js'></script>\n"
    "</body>\n"
    "</html>\n";

char* render_todoapp_page() {
    char* html = (char*)malloc(65536);
    int offset = 0;

    offset += sprintf(html + offset, html_header, "TodoApp");
    offset += sprintf(html + offset, "<h1 class='mb-4'>TodoApp</h1>\n");

    // DataList - Show all records
    offset += sprintf(html + offset, "<h2>All Items</h2>\n");
    offset += sprintf(html + offset, "<table class='table table-striped'>\n");
    offset += sprintf(html + offset, "<thead><tr>");
    offset += sprintf(html + offset, "<th>Id</th>");
    offset += sprintf(html + offset, "<th>Title</th>");
    offset += sprintf(html + offset, "<th>Description</th>");
    offset += sprintf(html + offset, "<th>Completed</th>");
    offset += sprintf(html + offset, "<th>Actions</th>");
    offset += sprintf(html + offset, "</tr></thead>\n");
    offset += sprintf(html + offset, "<tbody>\n");

    int count = 0;
    Todo** items = Todo_all(&count);
    for (int i = 0; i < count; i++) {
        offset += sprintf(html + offset, "<tr>");
        offset += sprintf(html + offset, "<td>%lld</td>", items[i]->id);
        offset += sprintf(html + offset, "<td>%s</td>", items[i]->title);
        offset += sprintf(html + offset, "<td>%s</td>", items[i]->description);
        offset += sprintf(html + offset, "<td>%s</td>", items[i]->completed ? "Yes" : "No");
        offset += sprintf(html + offset, "<td><a href='/todos/delete?id=%lld' class='btn btn-sm btn-danger'>Delete</a></td>", items[i]->id);
        offset += sprintf(html + offset, "</tr>\n");
    }

    offset += sprintf(html + offset, "</tbody></table>\n");

    // Form - Add new record
    offset += sprintf(html + offset, "<h2 class='mt-5'>Add New Item</h2>\n");
    offset += sprintf(html + offset, "<form method='POST' action='/todos/create'>\n");
    offset += sprintf(html + offset, "<div class='mb-3'>\n");
    offset += sprintf(html + offset, "<label class='form-label'>Title</label>\n");
    offset += sprintf(html + offset, "<input type='text' name='title' class='form-control' required>\n");
    offset += sprintf(html + offset, "</div>\n");
    offset += sprintf(html + offset, "<div class='mb-3'>\n");
    offset += sprintf(html + offset, "<label class='form-label'>Description</label>\n");
    offset += sprintf(html + offset, "<textarea name='description' class='form-control' rows='3' required></textarea>\n");
    offset += sprintf(html + offset, "</div>\n");
    offset += sprintf(html + offset, "<div class='mb-3'>\n");
    offset += sprintf(html + offset, "<label class='form-label'>Completed</label>\n");
    offset += sprintf(html + offset, "<input type='checkbox' name='completed' class='form-check-input'>\n");
    offset += sprintf(html + offset, "</div>\n");
    offset += sprintf(html + offset, "<button type='submit' class='btn btn-primary'>Add Item</button>\n");
    offset += sprintf(html + offset, "</form>\n");

    offset += sprintf(html + offset, "%s", html_footer);
    return html;
}


// URL decode helper
void url_decode(char *dst, const char *src) {
    char a, b;
    while (*src) {
        if ((*src == '%') && ((a = src[1]) && (b = src[2])) && (isxdigit(a) && isxdigit(b))) {
            if (a >= 'a') a -= 'a'-'A';
            if (a >= 'A') a -= ('A' - 10);
            else a -= '0';
            if (b >= 'a') b -= 'a'-'A';
            if (b >= 'A') b -= ('A' - 10);
            else b -= '0';
            *dst++ = 16*a+b;
            src+=3;
        } else if (*src == '+') {
            *dst++ = ' ';
            src++;
        } else {
            *dst++ = *src++;
        }
    }
    *dst++ = '\0';
}

// Parse form data
void parse_form_data(const char* data, char fields[][256], char values[][256], int* count) {
    char* datacopy = strdup(data);
    char* pair = strtok(datacopy, "&");
    *count = 0;

    while (pair != NULL && *count < 10) {
        char* eq = strchr(pair, '=');
        if (eq) {
            *eq = '\0';
            url_decode(fields[*count], pair);
            url_decode(values[*count], eq + 1);
            (*count)++;
        }
        pair = strtok(NULL, "&");
    }
    free(datacopy);
}


// HTTP request handler
enum MHD_Result handle_request(void *cls, struct MHD_Connection *connection,
                   const char *url, const char *method,
                   const char *version, const char *upload_data,
                   size_t *upload_data_size, void **con_cls) {

    struct MHD_Response *response;
    int ret;

    if (strcmp(url, "/todos/create") == 0 && strcmp(method, "POST") == 0) {
        // First call: set up
        if (*con_cls == NULL) {
            *con_cls = (void*)1;
            return MHD_YES;
        }

        // Process POST data
        if (*upload_data_size != 0) {
            char fields[10][256];
            char values[10][256];
            int count;
            parse_form_data(upload_data, fields, values, &count);

            // Extract form values
            char* title = "";
            char* description = "";
            int completed = 0;
            for (int i = 0; i < count; i++) {
                if (strcmp(fields[i], "title") == 0) title = strdup(values[i]);
                if (strcmp(fields[i], "description") == 0) description = strdup(values[i]);
                if (strcmp(fields[i], "completed") == 0) completed = 1;
            }

            Todo_create(title, description, completed);

            *upload_data_size = 0;
            return MHD_YES;
        }

        // Send redirect response
        const char* redirect = "<html><head><meta http-equiv='refresh' content='0;url=/'></head></html>";
        response = MHD_create_response_from_buffer(strlen(redirect), (void*)redirect, MHD_RESPMEM_PERSISTENT);
        ret = MHD_queue_response(connection, MHD_HTTP_SEE_OTHER, response);
        MHD_add_response_header(response, "Location", "/");
        MHD_destroy_response(response);
        return ret;
    }

    if (strncmp(url, "/todos/delete", 13) == 0 && strcmp(method, "GET") == 0) {
        const char* id_str = MHD_lookup_connection_value(connection, MHD_GET_ARGUMENT_KIND, "id");
        if (id_str) {
            int64_t id = atoll(id_str);
            Todo_delete(id);
        }
        const char* redirect = "<html><head><meta http-equiv='refresh' content='0;url=/'></head></html>";
        response = MHD_create_response_from_buffer(strlen(redirect), (void*)redirect, MHD_RESPMEM_PERSISTENT);
        ret = MHD_queue_response(connection, MHD_HTTP_OK, response);
        MHD_destroy_response(response);
        return ret;
    }

    if (strcmp(url, "/") == 0 && strcmp(method, "GET") == 0) {
        char* html = render_todoapp_page();
        response = MHD_create_response_from_buffer(strlen(html), (void*)html, MHD_RESPMEM_MUST_FREE);
        MHD_add_response_header(response, "Content-Type", "text/html");
        ret = MHD_queue_response(connection, MHD_HTTP_OK, response);
        MHD_destroy_response(response);
        return ret;
    }

    // 404
    const char* not_found = "<h1>404 Not Found</h1>";
    response = MHD_create_response_from_buffer(strlen(not_found), (void*)not_found, MHD_RESPMEM_PERSISTENT);
    ret = MHD_queue_response(connection, MHD_HTTP_NOT_FOUND, response);
    MHD_destroy_response(response);
    return ret;
}

int main(int argc, char *argv[]) {
    // Initialize database
    int rc = sqlite3_open("app.db", &db);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Cannot open database: %s\n", sqlite3_errmsg(db));
        return 1;
    }
    printf("Database opened successfully\n");

    Todo_init_table();

    // Start HTTP server
    http_daemon = MHD_start_daemon(MHD_USE_SELECT_INTERNALLY, 8080, NULL, NULL,
                                    &handle_request, NULL, MHD_OPTION_END);
    if (http_daemon == NULL) {
        fprintf(stderr, "Failed to start HTTP server\n");
        return 1;
    }

    printf("\n========================================\n");
    printf("Server running on http://localhost:8080\n");
    printf("Press ENTER to stop the server...\n");
    printf("========================================\n\n");

    getchar();

    // Stop HTTP server
    MHD_stop_daemon(http_daemon);

    // Close database
    sqlite3_close(db);
    printf("Server stopped\n");
    return 0;
}
