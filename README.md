# nyc-asp-lambda

nyc-asp-lambda (New York City Alternate Side Parking) is a AWS Lambda written in Go which is triggered daily by a AWS EventBridge scheduled rule which gets the NYC alternate side parking status and details from [NYC 311 Public API](https://api-portal.nyc.gov/api-details#api=nyc-311-public-api&operation=api-GetCalendar-get) and then publishes the details to an AWS SNS topic. The AWS SNS topic will maintain registered email or mobile subscriptions to consume the published events. 

## Purpose
The goal of the nyc-asp-lambda is to notify the topic subscribers of NYC alternate side parking days on which the status is suspended due to a observed holiday or special NYC reason. 

## Prerequistes 
* AWS Account
* AWS Lambda, SNS, EventBridge
* Go runtime development support


