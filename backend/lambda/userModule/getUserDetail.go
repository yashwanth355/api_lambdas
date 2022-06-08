//Deployed with pwd changes- Updated table names-Sep6
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

type UserDetails struct {
	Userid            string `json:"userid"`
	Firstname         string `json:"firstname"`
	Middlename        string `json:"middlename"`
	Lastname          string `json:"lastname"`
	Emailid           string `json:"emailid"`
	Alias             string `json:"alias"`
	Username          string `json:"username"`
	Empcode           string `json:"empcode"`
	Designation       string `json:"designation"`
	Company           string `json:"company"`
	Department        string `json:"department"`
	Role              string `json:"role"`
	Division          string `json:"division"`
	Ext               string `json:"ext"`
	Mobile            string `json:"mobile"`
	Phone             string `json:"phone"`
	Employee          bool   `json:"employee"`
	Profile           string `json:"profile"`
	Title             string `json:"title"`
	Active            bool   `json:"active"`
	Delegatedapprover string `json:"delegatedapprover"`
	Manager           string `json:"manager"`
	Street            string `json:"street"`
	State             string `json:"state"`
	Postalcode        string `json:"postalcode"`
	City              string `json:"city"`
	Country           string `json:"country"`
}

type User struct {
	Userid     string `json:"userid"`
	Firstname  string `json:"firstname"`
	Emailid    string `json:"emailid"`
	Department string `json:"department"`
	Role       string `json:"role"`
	Title      string `json:"title"`
	Active     bool   `json:"active"`
}

type UserInfo struct {
	EmailId string `json:"emailid"`
	Type    string `json:"type"`
	Active  bool   `json:"active"`
	Filter  string `json:"filter"`
}

func getUserDetail(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var userInfo UserInfo
	err := json.Unmarshal([]byte(request.Body), &userInfo)
	db, err := sql.Open("postgres", psqlInfo)

	if err != nil {
		log.Println(err)
	}
	defer db.Close()

	// check db
	err = db.Ping()

	if err != nil {
		log.Println(err)
	}

	fmt.Println("Connected!")

	if userInfo.Type == "getuser" {
		var userid, firstname, middlename, lastname, emailid, alias, username, empcode, designation, company, department, role, division, profile, delegatedapprover, manager, ext, phone, mobile, country, state, street, city, postalcode, title sql.NullString
		
		sqlStatement := `select userid, firstname, middlename, lastname, 
						emailid, alias, username, empcode, designation, 
						company, department, role, division,employee, 
						profile, delegatedapprover, manager, active, 
						street, state, postalcode, title, city, country, 
						ext, phone, mobile 
						from dbo.users_master_newpg
						where emailid=$1`

		rows, err := db.Query(sqlStatement, userInfo.EmailId)
		var user UserDetails
		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(&userid, &firstname, &middlename, &lastname, 
				&emailid, &alias, &username, &empcode, &designation, 
				&company, &department, &role, &division, &user.Employee,
				&profile, &delegatedapprover, &manager, &user.Active,
				&street,&state,&postalcode,&title,&city,&country,
				&ext, &phone, &mobile)
		}
		user.Userid=userid.String
		user.Firstname=firstname.String
		user.Middlename=middlename.String
		user.Lastname=lastname.String
		user.Emailid=emailid.String
		user.Alias=alias.String
		user.Username=username.String
		user.Empcode=empcode.String
		user.Designation=designation.String
		user.Company=company.String
		user.Department=department.String
		user.Role=role.String
		user.Division=division.String
		user.Profile=profile.String
		user.Delegatedapprover=delegatedapprover.String
		user.Manager=manager.String
		user.Street=street.String
		user.State=state.String
		user.Postalcode=postalcode.String
		user.Title=title.String
		user.City=city.String
		user.Country=country.String
		user.Ext=ext.String
		user.Phone=phone.String
		user.Mobile=mobile.String
		

		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		res, _ := json.Marshal(user)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil

	} else if userInfo.Type == "changestatus" {
		sqlStatement := `update dbo.users_master_newpg 
							set 
							active=$1 
							where emailid=$2`

		rows, err := db.Query(sqlStatement, userInfo.Active, userInfo.EmailId)
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		res, _ := json.Marshal(rows)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil

	} else {
		var param string
		if userInfo.Filter == "" {
			param = " order by idsno desc"
		} else {
			param = "where " + userInfo.Filter + " order by idsno desc"
		}

		log.Println("filter Query :", param)
		sqlStatement := `select userid, emailid, firstname, department, role, title, active from dbo.users_master_newpg %s`

		rows, err := db.Query(fmt.Sprintf(sqlStatement, param))
		var userDetailsList []User
		var firstName, department, role, title sql.NullString
		defer rows.Close()
		for rows.Next() {
			var user User
			err = rows.Scan(&user.Userid, &user.Emailid, &firstName, &department, &role, &title, &user.Active)
			user.Firstname = firstName.String
			user.Role = role.String
			user.Title = title.String
			user.Department = department.String
			userDetailsList = append(userDetailsList, user)
		}

		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		res, _ := json.Marshal(userDetailsList)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	}
}

func main() {
	lambda.Start(getUserDetail)
}
