package main

import (
	"bytes"
	"fmt"
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
	fmt.Println(data)
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
	start := time.Now().Add(time.Hour * -24)
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
		return dataRes.Datapoints[i].Timestamp.Before(*dataRes.Datapoints[j].Timestamp)
	})

	return dataRes.Datapoints, nil
}

// Create and output a graph as an image
func draw_graph(data []*cloudwatch.Datapoint) error {
	var x, y []float64
	for _, points := range data {
		x = append(x, -time.Since(*points.Timestamp).Hours())
		y = append(y, *points.Maximum)
		fmt.Println(time.Since(*points.Timestamp).Seconds(), "\n")
	}

	graph := chart.Chart{
		XAxis: chart.XAxis{
			Name:      "Hours ago",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Name:      "CPU Utilisation (average percent)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Background: chart.Style{
			Padding: chart.Box{
				Top:  20,
				Left: 20,
			},
		},
		Series: []chart.Series{
			chart.ContinuousSeries{
				Name:    "CPU Utilisation over time",
				XValues: x,
				YValues: y,
			},
		},
	}

	//note we have to do this as a separate step because we need a reference to graph
	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
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
