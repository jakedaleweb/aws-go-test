package main

import (
	"bytes"
	"log"
	"os"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	chart "github.com/wcharczuk/go-chart"
)

func main() {
	svc := start_service()
	data, err := get_data(svc)
	if err != nil {
		log.Fatal(err)
	}
	if err := draw_graph(data); err != nil {
		log.Fatal(err)
	}
}

// Create CloudWatch client
func start_service() *cloudwatch.CloudWatch {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	return cloudwatch.New(sess)
}

// Get data from CloudWatch and sort by date
func get_data(svc *cloudwatch.CloudWatch) ([]*cloudwatch.Datapoint, error) {
	end := time.Now()
	start := time.Now().AddDate(0, 0, -1)
	period := 60
	var i64 int64
	i64 = int64(period)
	max, min := "Maximum", "Minimum"

	dataRes, err := svc.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		MetricName: aws.String("CPUUtilization"),
		Namespace:  aws.String("AWS/EC2"),
		Dimensions: []*cloudwatch.Dimension{
			&cloudwatch.Dimension{
				Name:  aws.String("InstanceId"),
				Value: aws.String("i-0f73d203e5c90866f"),
			},
		},
		StartTime: &start,
		EndTime:   &end,
		Period:    &i64,
		Statistics: []*string{
			&max,
			&min,
		},
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(dataRes.Datapoints, func(i, j int) bool {
		return dataRes.Datapoints[i].Timestamp.After(*dataRes.Datapoints[j].Timestamp)
	})

	return dataRes.Datapoints, nil
}

func draw_graph(data []*cloudwatch.Datapoint) error {
	var x, y []float64
	for _, points := range data {
		x = append(x, time.Since(*points.Timestamp).Seconds())
		y = append(y, *points.Maximum)
	}

	graph := chart.Chart{
		Series: []chart.Series{
			chart.ContinuousSeries{
				XValues: x,
				YValues: y,
			},
		},
	}

	buffer := bytes.NewBuffer([]byte{})
	err := graph.Render(chart.PNG, buffer)

	if err != nil {
		return err
	}

	fo, err := os.Create("output.png")
	if err != nil {
		return err
	}

	if _, err := fo.Write(buffer.Bytes()); err != nil {
		return err
	}

	return nil
}
