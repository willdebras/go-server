// key-value storagepackage main

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi"
)

var StoragePath = "/tmp"

func main() {
	// Get port from env variables or set to 8080.
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}
	log.Printf("Starting up on http://localhost:%s", port)

	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, map[string]string{"key": "value"})
	})

	r.Get("/key/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := chi.URLParam(r, "key") // chi wont provide key for us, have to call it as var

		data, err := Get(r.Context(), key) // go returns both response and error always, if err not nil, we handle it
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)    // this equals 500
			JSON(w, map[string]string{"error": err.Error()}) // return JSON error
			return
		}

		w.Write([]byte(data)) // otherwise write our data
	})

	r.Delete("/key/{key}", func(w http.ResponseWriter, r *http.Request) { // now we add handler for if request is DELETE
		key := chi.URLParam(r, "key")

		err := Delete(r.Context(), key)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			JSON(w, map[string]string{"error": err.Error()})
			return
		}

		JSON(w, map[string]string{"status": "success"})
	})

	r.Post("/key/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := chi.URLParam(r, "key")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			JSON(w, map[string]string{"error": err.Error()})
			return
		}

		err = Set(r.Context(), key, string(body))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			JSON(w, map[string]string{"error": err.Error()})
			return
		}

		JSON(w, map[string]string{"status": "success"})
	})

	log.Fatal(http.ListenAndServe(":"+port, r))
}

func JSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	b, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]string{"error": err.Error()})
		return
	}

	w.Write(b)
}

func Get(ctx context.Context, key string) (string, error) {
	data, err := loadData(ctx)
	if err != nil {
		return "", err
	}

	return data[key], nil
}

func Set(ctx context.Context, key string, value string) error {
	data, err := loadData(ctx)
	if err != nil {
		return err
	}

	data[key] = value
	if err := saveData(ctx, data); err != nil {
		return err
	}

	return nil
}

func Delete(ctx context.Context, key string) error {
	data, err := loadData(ctx)
	if err != nil {
		return err
	}

	delete(data, key)
	return nil
}

func dataPath() string {
	return filepath.Join(StoragePath, "data.json")
}

func loadData(ctx context.Context) (map[string]string, error) {
	empty := map[string]string{}
	emptyData, err := encode(map[string]string{})
	if err != nil {
		return empty, err
	}

	// First check if the folder exists and create it if it is missing.
	if _, err := os.Stat(StoragePath); os.IsNotExist(err) {
		err = os.MkdirAll(StoragePath, 0755)
		if err != nil {
			return empty, err
		}
	}

	// Then check if the file exists and create it if it is missing.
	if _, err := os.Stat(dataPath()); os.IsNotExist(err) {
		err := os.WriteFile(dataPath(), emptyData, 0644)
		if err != nil {
			return empty, err
		}
	}

	content, err := os.ReadFile(dataPath())
	if err != nil {
		return empty, err
	}

	return decode(content)
}

func saveData(ctx context.Context, data map[string]string) error {
	// First check if the folder exists and create it if it is missing.
	if _, err := os.Stat(StoragePath); os.IsNotExist(err) {
		err = os.MkdirAll(StoragePath, 0755)
		if err != nil {
			return err
		}
	}

	encodedData, err := encode(data)
	if err != nil {
		return err
	}

	return os.WriteFile(dataPath(), encodedData, 0644)
}

func encode(data map[string]string) ([]byte, error) {
	encodedData := map[string]string{}
	for k, v := range data {
		ek := base64.URLEncoding.EncodeToString([]byte(k))
		ev := base64.URLEncoding.EncodeToString([]byte(v))
		encodedData[ek] = ev
	}

	return json.Marshal(encodedData)
}

func decode(data []byte) (map[string]string, error) {
	var jsonData map[string]string

	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, err
	}

	returnData := map[string]string{}
	for k, v := range jsonData {
		dk, err := base64.URLEncoding.DecodeString(k)
		if err != nil {
			return nil, err
		}

		dv, err := base64.URLEncoding.DecodeString(v)
		if err != nil {
			return nil, err
		}

		returnData[string(dk)] = string(dv)
	}

	return returnData, nil
}
