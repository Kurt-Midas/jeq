package main

import (
	"fmt"
	"github.com/kurt-midas/go-crest/crest"
	"html/template"
	"net/http"
	"strconv"
)

// 41040 - stasis grappler
// 41054 - heavy gunnar
// 41056 - heavy jigoro
// 41055 - heavy karelin
// 9668 - Large Concussion
//
var itemlist = []int{41056, 41054, 41055,
	7451, 7447, 7449, 7453,
	14282, 14280, 14284, 14286,
	7409, 7411, 7413, 7404,
	5945, 35662, 35661, //500mn
	5975, 35660, 35659, //50mn
	25660, 25599, 25604, 25597, 25601, 25589, 25588, 25598, 25606, //salvage
				35657, 5955} //100mn
var jitaID int = 10000002 //?
var valeID int = 10000038 //??
var pricePerM3 float64 = 500

var brokerFee float32 = 3.0
var salesTax float32 = 2.0

func main() {
	// fmt.Printf("Printing template data :: %+v\n", populateTemplateData())
	// populateTemplateData()
	// http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/", indexBuilder)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./bower_components"))))
	fmt.Println("JEQ is Listening on 8081...")
	http.ListenAndServe(":8081", nil)
}

func indexBuilder(w http.ResponseWriter, r *http.Request) {
	fmt.Println("How does one pronounce JEQ?")
	data := populateTemplateData()
	t, err := template.New("index.html").Delims("[[", "]]").ParseFiles("./frontend/index.html")
	if err != nil {
		panic(err)
	}
	t.Execute(w, data)
}

type templateData struct {
	ItemDetails   map[string]myItemDetails
	MarketOrders  map[string]orderDataSystems   //types
	MarketHistory map[string]historyDataSystems //types
}

type historyDataSystems struct {
	Jita  historyData `json:"jita"`
	Other historyData `json:"other"`
}

type historyData struct {
	Typeid    int                       //do not jsonify
	AvgPrice  float64                   `json:"avgPrice"`
	AvgVolume float64                   `json:"avgVolume"`
	AvgOrders float64                   `json:"avgOrders"`
	Days      []crest.MarketTypeHistory `json:"days"`
}

type orderDataSystems struct {
	Jita  orderData `json:"jita"`
	Other orderData `json:"other"`
}

type orderData struct {
	Buy struct {
		Orders    []crest.MarketOrderType `json:"orders"`
		BestPrice float64                 `json:"bestPrice"`
		Volume    int                     `json:"volume"`
	} `json:"buy"`
	Sell struct {
		Orders    []crest.MarketOrderType `json:"orders"`
		BestPrice float64                 `json:"bestPrice"`
		Volume    int                     `json:"volume"`
	} `json:"sell"`
}

type myItemDetails struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Volume float64 `json:"volume"`
}

func populateOrderData(orders []crest.MarketOrderType) (orderData, int) {
	info := orderData{}
	var typeid int = 0
	for _, order := range orders {
		if order.Buy {
			// fmt.Printf("Buy order %v for type %v\n", order.Id, order.Type.Id)
			if order.Price > info.Buy.BestPrice && order.Location.Id == 60003760 { //moon 4
				info.Buy.BestPrice = order.Price
				info.Buy.Volume += order.Volume
			}
			info.Buy.Orders = append(info.Buy.Orders, order)
		} else {
			// fmt.Printf("Sell order %v for type %v\n", order.Id, order.Type.Id)
			if (order.Price < info.Sell.BestPrice || info.Sell.BestPrice == 0) && order.Location.Id == 60003760 {
				info.Sell.BestPrice = order.Price
				info.Sell.Volume += order.Volume
			}
			info.Sell.Orders = append(info.Sell.Orders, order)
		}
		typeid = order.Type.Id
	}
	return info, typeid
}

func populateTemplateData() templateData {
	// ch := make(chan *templateData)
	data := templateData{}
	data.ItemDetails = make(map[string]myItemDetails)
	data.MarketHistory = make(map[string]historyDataSystems)
	data.MarketOrders = make(map[string]orderDataSystems)

	chItemDetails := make(chan *[]myItemDetails)
	chJitaOrders := make(chan *[][]crest.MarketOrderType)
	chValeOrders := make(chan *[][]crest.MarketOrderType)
	chJitaHistory := make(chan *[]myMarketTypeHistory)
	chValeHistory := make(chan *[]myMarketTypeHistory)
	go func(types []int) {
		details := getItemDetails(types)
		chItemDetails <- &details
	}(itemlist)
	go func(types []int, region int) {
		orders := getMarketOrders(types, region)
		chJitaOrders <- &orders
	}(itemlist, jitaID)
	go func(types []int, region int) {
		orders := getMarketOrders(types, region)
		chValeOrders <- &orders
	}(itemlist, valeID)
	go func(types []int, region int) {
		history := getMarketHistory(types, region)
		chJitaHistory <- &history
	}(itemlist, jitaID)
	go func(types []int, region int) {
		history := getMarketHistory(types, region)
		chValeHistory <- &history
	}(itemlist, valeID)
	mapJitaOrders := make(map[string]orderData)
	mapValeOrders := make(map[string]orderData)
	mapJitaHistory := make(map[string]historyData)
	mapValeHistory := make(map[string]historyData)
	for i := 0; i < 5; i++ {
		select {
		case r := <-chItemDetails:
			fmt.Printf("Hello %v\n", len(*r))
			for _, details := range *r {
				data.ItemDetails[strconv.Itoa(details.ID)] = details
			}
		case r := <-chJitaOrders:
			fmt.Printf("Jita Orders case : %v\n", len(*r))
			for _, orders := range *r {
				info, typeid := populateOrderData(orders)
				mapJitaOrders[strconv.Itoa(typeid)] = info
			}
		case r := <-chValeOrders:
			fmt.Printf("Vale Orders case %v\n", len(*r))
			for _, orders := range *r {
				info, typeid := populateOrderData(orders)
				mapValeOrders[strconv.Itoa(typeid)] = info
			}
		case r := <-chJitaHistory:
			fmt.Printf("Jita History case : %v\n", len(*r))
			for _, history := range populateHistoryData(*r) {
				mapJitaHistory[strconv.Itoa(history.Typeid)] = history
			}
		case r := <-chValeHistory:
			fmt.Printf("Vale History case : %v\n", len(*r))
			for _, history := range populateHistoryData(*r) {
				mapValeHistory[strconv.Itoa(history.Typeid)] = history
			}
		} //select
	}
	for key := range data.ItemDetails {
		data.MarketOrders[key] = orderDataSystems{mapJitaOrders[key], mapValeOrders[key]}
		data.MarketHistory[key] = historyDataSystems{mapJitaHistory[key], mapValeHistory[key]}
	}
	return data
}

func populateHistoryData(r []myMarketTypeHistory) []historyData {
	// fmt.Printf("Jita History case : %v\n", r)
	data := make([]historyData, 0)
	for _, typeHistory := range r {
		if len(typeHistory.History) < 30 {
			fmt.Printf("History had less than 30 days :: ", typeHistory)
		} else {
			typeHistory.History = typeHistory.History[len(typeHistory.History)-30:]
		}
		// fmt.Printf("Length of typehistory is %v\n", len(typeHistory.History))
		var avgPrice float64 = 0.0
		var avgOrderCount float64 = 0.0
		var avgVolume float64 = 0.0
		for _, day := range typeHistory.History {
			avgPrice += day.AvgPrice
			avgOrderCount += float64(day.OrderCount)
			avgVolume += float64(day.Volume)
		}
		data = append(data, historyData{typeHistory.TypeID, avgPrice / 30, avgVolume / 30, avgOrderCount / 30, typeHistory.History})
	}
	return data
}

func getItemDetails(types []int) []myItemDetails {
	fmt.Println("Inside getItemDetails")
	//turn each type into a details group with type, name, volume, and groups
	ch := make(chan *crest.InvTypeDetails)
	detailsList := []myItemDetails{}
	for _, typeid := range types {
		go func(id int) {
			details, err := crest.InventoryType(id)
			if err != nil {
				fmt.Printf("Err at id %v :: %v\n", id, err)
			}
			ch <- &details
		}(typeid)
	}
	for {
		select {
		case r := <-ch:
			fmt.Printf("Response included %v, %v, %v\n", r.ID, r.Name, r.Volume)
			detailsList = append(detailsList, myItemDetails{r.ID, r.Name, r.Volume})
			if len(detailsList) == len(types) {
				return detailsList
			}
		}
	}
}

func getMarketOrders(types []int, region int) [][]crest.MarketOrderType {
	fmt.Println("Inside getMarketOrders")
	ch := make(chan *[]crest.MarketOrderType)
	detailsList := [][]crest.MarketOrderType{}
	for _, typeid := range types {
		go func(id int) {
			orders, err := crest.MarketOrders_Type_All(region, id)
			if err != nil {
				fmt.Printf("Err at id %v :: %v\n", id, err)
			}
			ch <- &orders
		}(typeid)
	}
	for {
		select {
		case r := <-ch:
			// fmt.Printf("Response included %v, %v, %v\n", r.Id, r.Price, r.Volume)
			detailsList = append(detailsList, *r)
			if len(detailsList) == len(types) {
				return detailsList
			}
		}
	}
}

type myMarketTypeHistory struct {
	History []crest.MarketTypeHistory
	TypeID  int
}

func getMarketHistory(types []int, region int) []myMarketTypeHistory {
	fmt.Println("Inside getMarketHistory")
	ch := make(chan *myMarketTypeHistory)
	detailsList := []myMarketTypeHistory{}
	for _, typeid := range types {
		go func(id int) {
			orders, err := crest.MarketHistory_Type(region, id)
			if err != nil {
				fmt.Printf("Err at id %v :: %v\n", id, err)
			}
			ch <- &myMarketTypeHistory{orders, id}
		}(typeid)
	}
	for {
		select {
		case r := <-ch:
			// fmt.Printf("Response included %v, %v, %v\n", r.Id, r.Price, r.Volume)
			detailsList = append(detailsList, *r)
			if len(detailsList) == len(types) {
				return detailsList
			}
		}
	}
}
