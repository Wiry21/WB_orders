package main

import (
	"encoding/json"
	"html/template"
	"l0/json_struct"
	"net/http"

	"context"
	"fmt"
	"log"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
	stan "github.com/nats-io/stan.go"
)

var Cache = make(map[string]json_struct.Order)

func ReadBytes(m *stan.Msg) error {
	var order json_struct.Order
	err := json.Unmarshal(m.Data, &order)
	if err != nil {
		return err
	}
	if order.OrderUid == "" {
		return fmt.Errorf("No OrderUid, OrderUid lost?")
	}

	Cache[order.OrderUid] = order

	ctx := context.Background()
	databaseUrl := "postgres://Wiry:postgres@localhost:5432/Wiry"
	dbpool, err := pgxpool.Connect(ctx, databaseUrl)
	if err != nil {
		return fmt.Errorf("Unable to connect to database: %v\n", err)
	}
	defer dbpool.Close()

	queryDelete := `delete from payments where transaction = $1;`
	_, err = dbpool.Exec(ctx, queryDelete, "b563feb7b2b84b6test")
	if err != nil {
		fmt.Printf("Unable to delete: %v\n", err)
	}
	queryDelete = `delete from orders where order_uid = $1;`
	_, err = dbpool.Exec(ctx, queryDelete, "b563feb7b2b84b6test")
	if err != nil {
		fmt.Printf("Unable to delete: %v\n", err)
	}
	queryDelete = `delete from deliveries where customer_id = $1;`
	_, err = dbpool.Exec(ctx, queryDelete, "test")
	if err != nil {
		fmt.Printf("Unable to delete: %v\n", err)
	}
	queryDelete = `delete from items where item_id = $1;`
	_, err = dbpool.Exec(ctx, queryDelete, "b563feb7b2b84b6test")
	if err != nil {
		fmt.Printf("Unable to delete: %v\n", err)
	}

	queryInsert := `insert into orders
			(
				order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey,
				sm_id, date_created, oof_shard
			)
			values
			(
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
			);`
	_, err = dbpool.Exec(ctx, queryInsert,
		order.OrderUid, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature, order.CustomerId,
		order.DeliveryService, order.Shardkey, order.SmId, order.DateCreated, order.OofShard)
	if err != nil {
		dbpool.Exec(ctx, `delete from orders where order_uid = $1`, order.OrderUid)
		return fmt.Errorf("Unable to insert: %v\n", err)
	}
	queryInsert = `insert into deliveries
			(
				customer_id, name, phone, zip, city, address, region, email
			)
			values
			(
				$1, $2, $3, $4, $5, $6, $7, $8
			);`
	_, err = dbpool.Exec(ctx, queryInsert,
		order.CustomerId, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		dbpool.Exec(ctx, `delete from orders where order_uid = $1`, order.OrderUid)
		return fmt.Errorf("Unable to insert: %v\n", err)
	}
	queryInsert = `insert into payments
			(
				payment_id, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost,
		        goods_total, custom_fee
			)
			values
			(
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
			);`
	_, err = dbpool.Exec(ctx, queryInsert,
		order.OrderUid, order.Payment.Transaction, order.Payment.RequestId, order.Payment.Currency, order.Payment.Provider,
		order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank, order.Payment.DeliveryCost,
		order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		dbpool.Exec(ctx, `delete from orders where order_uid = $1`, order.OrderUid)
		return fmt.Errorf("Unable to insert: %v\n", err)
	}
	queryInsert = `insert into items
			(
				item_id, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
			)
			values
			(
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
			);`
	for i := 0; i < len(order.Items); i++ {
		_, err = dbpool.Exec(ctx, queryInsert,
			order.OrderUid, order.Items[i].ChrtId, order.Items[i].TrackNumber, order.Items[i].Price, order.Items[i].Rid,
			order.Items[i].Name, order.Items[i].Sale, order.Items[i].Size, order.Items[i].TotalPrice, order.Items[i].NmId, order.Items[i].Brand, order.Items[i].Status)
		if err != nil {
			dbpool.Exec(ctx, `delete from orders where order_uid = $1`, order.OrderUid)
			return fmt.Errorf("Unable to insert: %v\n", err)
		}
	}
	fmt.Println("Successfully insert")
	return err
}

func ReadCacheFromDb() {
	ctx := context.Background()
	databaseUrl := "postgres://Wiry:postgres@localhost:5432/Wiry"
	dbpool, err := pgxpool.Connect(ctx, databaseUrl)
	defer dbpool.Close()
	if err != nil {
		log.Println(err.Error())
		return
	}

	var order []*json_struct.Order
	var delivery []*json_struct.Delivery
	var payment []*json_struct.Payment
	var item []*json_struct.Items

	querySelect := `select order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service,
	shardkey, sm_id, date_created, oof_shard from orders`
	err = pgxscan.Select(ctx, dbpool, &order, querySelect)
	if err != nil {
		log.Println(err.Error())
	}

	querySelect = `select name, phone, zip, city, address, region, email from deliveries`
	err = pgxscan.Select(ctx, dbpool, &delivery, querySelect)
	if err != nil {
		log.Println(err.Error())
	}

	querySelect = `select transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost,
	goods_total, custom_fee from payments`
	err = pgxscan.Select(ctx, dbpool, &payment, querySelect)
	if err != nil {
		log.Println(err.Error())
	}

	querySelect = `select chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status from items`
	err = pgxscan.Select(ctx, dbpool, &item, querySelect)
	if err != nil {
		log.Println(err.Error())
	}

	var itemTracks []string
	rows, err := dbpool.Query(ctx, `select track_number from items`)
	if err != nil {
		log.Println(err.Error())
	}
	for rows.Next() {
		var itemId string
		err = rows.Scan(&itemId)
		if err != nil {
			log.Println(err.Error())
		}
		itemTracks = append(itemTracks, itemId)
	}

	j := 0
	for i := 0; i < len(order); i++ {
		order[i].Delivery = *delivery[i]
		order[i].Payment = *payment[i]
		for j < len(itemTracks) {
			var itemStruct []json_struct.Items
			if itemTracks[j] == order[i].TrackNumber {
				itemStruct = append(itemStruct, *item[j])
				j++
				order[i].Items = itemStruct
			}
		}
		Cache[order[i].OrderUid] = *order[i]
		//		fmt.Println("map:", Cache)
	}
	fmt.Println("Загрузка кэша из бд завершена успешно")
}

func HomePage(w http.ResponseWriter, r *http.Request) {
	tmp, err := template.ParseFiles("templates/server.html")
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal server error", 500)
		return
	}
	err = tmp.Execute(w, nil)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal server error", 500)
		return
	}
}

func OrderById(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if _, exist := Cache[id]; exist {
		b, _ := json.Marshal(Cache[id])
		_, err := w.Write(b)
		if err != nil {
			log.Println(err.Error())
		}
	} else {
		_, err := w.Write([]byte("Запись не найдена"))
		if err != nil {
			log.Println(err.Error())
		}
	}
}

func OrderList(w http.ResponseWriter, r *http.Request) {
	orders := make([]json_struct.Order, 0)
	for _, elem := range Cache {
		orders = append(orders, elem)
	}

	b, _ := json.Marshal(orders)
	_, err := w.Write(b)
	if err != nil {
		log.Println(err.Error())
	}
}

func sub() {
	clusterID := "test-cluster" // nats cluster id
	clientID := "test-client"

	// Connect to NATS
	sc, err := stan.Connect(clusterID, clientID)
	if err != nil {
		log.Fatal(err)
	}

	subj, i := "subj", 0

	_, err = sc.Subscribe(subj, func(msg *stan.Msg) {
		err := ReadBytes(msg)
		if err == nil {
			log.Println("Received a message")
			i += 1
		} else {
			log.Println(err.Error())
		}
	})
	if err != nil {
		log.Println(err.Error())
	}

	log.Printf("Listening on [%s]", subj)
}

func main() {
	ReadCacheFromDb()

	mux := http.NewServeMux()
	mux.HandleFunc("/", HomePage)
	mux.HandleFunc("/order", OrderById)
	mux.HandleFunc("/list/", OrderList)

	go sub()

	log.Println("Запуск сервера...")
	log.Fatal(http.ListenAndServe(":8000", mux))
}
