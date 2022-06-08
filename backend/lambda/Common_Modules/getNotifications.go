package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
)

const (
	host     = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port     = 5432
	user     = "postgres"
	password = "Ccl_RDS_DB#2022"
	dbname   = "ccldevdb"
)

type Notify struct {
	
	// Type				string `json:"feature_type"`
	Status				string `json:"status"`
	Feature_Category	string `json:"feature_category"`
	Notid				string `json:"notification_id"`
	
}
type Input struct{
	ReadAll			  bool 		`json:"read_all"`
	View			  bool 		`json:"view"`
	GetCount		  bool 		`json:"get_count"`
	UserId  		  string 	`json:"userid"`
	Notify_Count	  string 	`json:"notify_count"`
	Notifications	 []Notify	`json:"notifications_feed"`
}


func getNotifications(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var input Input
	
	err := json.Unmarshal([]byte(request.Body), &input)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	defer db.Close()

	// check db
	err = db.Ping()

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	fmt.Println("Connected!")

	var rows *sql.Rows
	if input.UserId != "" {
		log.Println("Eneted notification module")
		if input.GetCount {
			
			sqlStatementC1 := `SELECT count(notid) FROM dbo.notifications_master_newpg where userid=$1 and readstatus=false`
			rows, err = db.Query(sqlStatementC1, input.UserId)
			for rows.Next() {
				err = rows.Scan(&input.Notify_Count)
			}
			log.Println("Total Count: ",input.Notify_Count)	
			res, _ := json.Marshal(input)
			return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
		} else if input.View {
			sqlStatement := `SELECT status, feature_category, notid
							FROM dbo.notifications_master_newpg where userid=$1 order by notid desc`
			rows, err = db.Query(sqlStatement, input.UserId)
			var notif Notify
			defer rows.Close()
			for rows.Next() {
				err = rows.Scan(&notif.Status,&notif.Feature_Category,&notif.Notid)
				allNotifs := append(input.Notifications,notif)
				input.Notifications=allNotifs
			}


			res, _ := json.Marshal(input)
			return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
		} else if input.ReadAll {
			log.Println("Mark all notifications as read")
			sqlStatementR1 := `update dbo.notifications_master_newpg 
							set
							readstatus=true
							where userid=$1`
			rows, err = db.Query(sqlStatementR1, input.UserId)
		}
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		
		
	}

	

	
	
	return events.APIGatewayProxyResponse{200, headers, nil, string("Success"), false}, nil
}

func main() {
	lambda.Start(getNotifications)
}