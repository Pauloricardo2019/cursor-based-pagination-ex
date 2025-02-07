package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var mapPaginationKey = sync.Map{}

const pageSize = 5
const deadLineCursorTime = 30 //seconds

type Products struct {
	ID       uint    `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity uint    `json:"quantity"`
}

type CursorPagination struct {
	Key       string
	Page      int
	CreatedAt time.Time
}

type GenericPagination[T any] struct {
	Content       []T    `json:"content"`
	CurrentPage   int    `json:"current_page"`
	TotalPages    int    `json:"total_pages"`
	NextCursorKey string `json:"next_cursor_key,omitempty"`
}

func generatedProducts() []Products {
	return []Products{
		{1, "Produto A", 10.99, 5},
		{2, "Produto B", 15.50, 10},
		{3, "Produto C", 7.30, 8},
		{4, "Produto D", 20.99, 3},
		{5, "Produto E", 5.49, 12},
		{6, "Produto F", 13.75, 7},
		{7, "Produto G", 8.99, 6},
		{8, "Produto H", 25.00, 4},
		{9, "Produto I", 18.20, 9},
		{10, "Produto J", 30.99, 2},
		{11, "Produto K", 22.49, 11},
		{12, "Produto L", 12.99, 14},
		{13, "Produto M", 16.75, 5},
		{14, "Produto N", 9.99, 8},
		{15, "Produto O", 27.99, 6},
		{16, "Produto P", 14.50, 7},
		{17, "Produto Q", 19.99, 3},
		{18, "Produto R", 24.75, 10},
		{19, "Produto S", 11.30, 9},
		{20, "Produto T", 21.49, 4},
		{21, "Produto U", 17.25, 13},
		{22, "Produto V", 29.99, 2},
		{23, "Produto W", 6.89, 15},
	}
}

func main() {

	serverMux := http.NewServeMux()

	go func() {
		fmt.Println("Inicializando go routine")
		timeTicker := time.NewTicker(time.Second * deadLineCursorTime)
		defer timeTicker.Stop()

		for {
			select {
			case <-timeTicker.C:

				mapPaginationKey.Range(func(key, value any) bool {
					k := key
					v := value.(CursorPagination)

					fmt.Printf("Dentro do loop, key: %s\n", key.(string))

					if !time.Now().Before(v.CreatedAt) {
						fmt.Printf("removing this, the times has finished, key: %s\n", k.(string))
						mapPaginationKey.Delete(k)
					}

					return true
				})

			}
		}

	}()

	serverMux.Handle("/todos", paginationValidator(http.HandlerFunc(getTodos)))
	fmt.Println("http://localhost:8080")
	http.ListenAndServe(":8080", serverMux)
}

func getTodos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	paginationKey := r.Header.Get("paginationKey")
	nextPageKey := r.Header.Get("nextPageKey")
	currentPage := r.Header.Get("currentPage")

	if nextPageKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	page := 0

	if paginationKey != "" {

		currentPageInt, err := strconv.Atoi(currentPage)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		page = currentPageInt
	}

	products := generatedProducts()

	totalPage := len(products) / pageSize
	offset := page * pageSize
	limit := offset + pageSize

	if limit > len(products) {
		limit = len(products)
	}

	pageProducts := products[offset:limit]

	if totalPage == page {
		nextPageKey = ""
	}

	genericPagination := GenericPagination[Products]{
		Content:       pageProducts,
		TotalPages:    totalPage + 1,
		CurrentPage:   page + 1,
		NextCursorKey: nextPageKey,
	}

	jBytes, err := json.MarshalIndent(genericPagination, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jBytes)
	return
}

func paginationValidator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		paginationKey := r.URL.Query().Get("paginationKey")

		r.Header.Set("paginationKey", paginationKey)

		if paginationKey == "" {

			nextPageKey := uuid.New().String()
			pageNumber := 1
			mapPaginationKey.Store(nextPageKey, CursorPagination{Key: nextPageKey, Page: pageNumber, CreatedAt: time.Now().Add(time.Second * deadLineCursorTime)})
			r.Header.Set("nextPageKey", nextPageKey)

			next.ServeHTTP(w, r)
			return
		}

		pageNumber, ok := mapPaginationKey.Load(paginationKey)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cursorPagination := pageNumber.(CursorPagination)

		nextPageKey := uuid.New().String()

		mapPaginationKey.Store(nextPageKey, CursorPagination{Key: nextPageKey, Page: cursorPagination.Page + 1, CreatedAt: time.Now().Add(time.Second * deadLineCursorTime)})

		r.Header.Set("nextPageKey", nextPageKey)
		r.Header.Set("currentPage", strconv.Itoa(cursorPagination.Page))

		next.ServeHTTP(w, r)
	})
}
