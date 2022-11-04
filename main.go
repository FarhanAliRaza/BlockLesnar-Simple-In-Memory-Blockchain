package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type createUser struct {
	name string
}

type Block struct {
	Index     int    `json:"index"`
	Timestamp string `json:"timestamp"`
	Uid       string `json:"uid"`
	Name      string `json:"name"`
	Amount    int    `json:"amount"`
	Is_owner  bool   `json:"is_owner"`
	Hash      string `json:"hash"`
	PrevHash  string `json:"prevHash"`
}

type Trasaction struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount int    `json:"amount"`
}

type TransactionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

var Blockchain []Block

func calculateHash(block Block) string {

	record := fmt.Sprintf("%v", block.Index) + block.Timestamp + fmt.Sprintf("%v", block.Uid) + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// Create a new user block

func createUserBlock(oldBlock Block, user createUser) (Block, error) {
	var newBlock Block
	t := time.Now()
	name := user.name
	id := uuid.New()
	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.Uid = id.String()
	newBlock.Name = name
	newBlock.Amount = 0
	newBlock.Is_owner = false
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = calculateHash(newBlock)

	return newBlock, nil
}
func createGensisUserBlock() (Block, error) {
	var newBlock Block
	t := time.Now()
	name := "farhan"
	id := uuid.New()
	newBlock.Index = 0
	newBlock.Timestamp = t.String()
	newBlock.Uid = id.String()
	newBlock.Name = name
	newBlock.Amount = 1000
	newBlock.Is_owner = true
	newBlock.PrevHash = ""
	newBlock.Hash = ""
	return newBlock, nil
}

func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
}

func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {

	bytes, err := json.Marshal(Blockchain)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// w.WriteHeader(http.StatusAccepted)
	w.Header().Set("Content-Type", "application/json")

	w.Write(bytes)
}

func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
	response, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}
	w.WriteHeader(code)
	w.Write(response)
}

func computeTransaction(t Trasaction) TransactionResponse {
	fromIndex := -1
	toIndex := -1
	for i := 0; i < len(Blockchain); i++ {

		if Blockchain[i].Uid == t.From {
			if Blockchain[i].Amount < t.Amount {
				return TransactionResponse{false, "Insufficient balance"}
			} else {
				fromIndex = i
			}
		}
		if Blockchain[i].Uid == t.To {
			toIndex = i
		}
	}
	if fromIndex != -1 && toIndex != -1 {
		Blockchain[fromIndex].Amount = Blockchain[fromIndex].Amount - t.Amount
		Blockchain[toIndex].Amount = Blockchain[toIndex].Amount + t.Amount
		return TransactionResponse{true, "Transaction successfull"}
	}
	return TransactionResponse{false, "Transaction failed"}

}

func handleTransaction(w http.ResponseWriter, r *http.Request) {
	var t Trasaction

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&t); err != nil {
		respondWithJSON(w, r, http.StatusBadRequest, r.Body)
		return
	}
	defer r.Body.Close()
	fmt.Println(t, "coming data")

	transaction := computeTransaction(t)
	fmt.Println(transaction, "transaction")
	w.Header().Set("Content-Type", "application/json")
	respondWithJSON(w, r, http.StatusAccepted, transaction)

}

func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var u createUser

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&u); err != nil {
		respondWithJSON(w, r, http.StatusBadRequest, r.Body)
		return
	}
	defer r.Body.Close()

	newBlock, err := createUserBlock(Blockchain[len(Blockchain)-1], u)
	if err != nil {
		respondWithJSON(w, r, http.StatusInternalServerError, u)
		return
	}
	if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
		newBlockchain := append(Blockchain, newBlock)
		replaceChain(newBlockchain)
		spew.Dump(Blockchain)
	}

	respondWithJSON(w, r, http.StatusCreated, newBlock)

}

func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/create/", handleCreateUser).Methods("POST")
	muxRouter.HandleFunc("/transact/", handleTransaction).Methods("POST")

	return muxRouter
}
func run() error {
	mux := makeMuxRouter()
	httpAddr := os.Getenv("ADDR")
	log.Println("Listening on ", os.Getenv("ADDR"))
	s := &http.Server{
		Addr:           "localhost:" + httpAddr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {

		genesisBlock, _ := createGensisUserBlock()
		spew.Dump(genesisBlock)
		Blockchain = append(Blockchain, genesisBlock)

	}()
	log.Fatal(run())

}
