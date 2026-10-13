#include <gtk/gtk.h>
#include <webkit/webkit.h>
#include "webview.h"

extern void goWindowClosed(void);

static GtkApplication *app;
static GtkWidget *win;
static char *page_url;
static bool show_at_start, started;

// Closing only hides the window, so reopening it is instant and keeps the page's state.
static gboolean on_close_request(GtkWindow *w, gpointer data) {
	gtk_widget_set_visible(GTK_WIDGET(w), FALSE);
	goWindowClosed();
	return TRUE;
}

static void present(void) {
	if (!win) {
		win = gtk_application_window_new(app);
		gtk_window_set_title(GTK_WINDOW(win), "Bulp");
		gtk_window_set_default_size(GTK_WINDOW(win), 389, 560);
		GtkWidget *web = webkit_web_view_new();
		webkit_web_view_load_uri(WEBKIT_WEB_VIEW(web), page_url);
		gtk_window_set_child(GTK_WINDOW(win), web);
		g_signal_connect(win, "close-request", G_CALLBACK(on_close_request), NULL);
	}
	gtk_window_present(GTK_WINDOW(win));
}

// Runs on our own start and whenever the app is launched again.
static void on_activate(GApplication *a, gpointer data) {
	if (!started) {
		started = true;
		if (!show_at_start) return;
	}
	present();
}

// wv_init returns 0 if GTK can't start, 1 for the primary instance, and 2 if
// another instance already runs (it has been asked to show its window).
int wv_init(const char *app_id) {
	g_set_application_name("Bulp");
	if (!gtk_init_check()) return 0;
	app = gtk_application_new(app_id, G_APPLICATION_DEFAULT_FLAGS);
	g_signal_connect(app, "activate", G_CALLBACK(on_activate), NULL);
	GError *err = NULL;
	if (!g_application_register(G_APPLICATION(app), NULL, &err)) {
		g_printerr("can't register %s: %s\n", app_id, err->message);
		g_error_free(err);
		return 0;
	}
	if (g_application_get_is_remote(G_APPLICATION(app))) {
		g_application_run(G_APPLICATION(app), 0, NULL); // activates the primary and returns
		return 2;
	}
	return 1;
}

void wv_run(const char *url, bool show) {
	page_url = g_strdup(url);
	show_at_start = show;
	g_application_hold(G_APPLICATION(app)); // keep running while only the tray icon is up
	g_application_run(G_APPLICATION(app), 0, NULL);
}

static gboolean do_show(gpointer data) {
	present();
	return G_SOURCE_REMOVE;
}

void wv_show(void) { g_idle_add(do_show, NULL); }

static gboolean do_quit(gpointer data) {
	g_application_quit(G_APPLICATION(app));
	return G_SOURCE_REMOVE;
}

void wv_quit(void) { g_idle_add(do_quit, NULL); }
