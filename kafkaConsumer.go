package main

import (
	sarama "github.com/Shopify/sarama"
	goavro "github.com/linkedin/goavro"
	"flag"
	"os"
	"crypto/tls"
	"log"
	"io/ioutil"
	"crypto/x509"
	"fmt"
	//"encoding/json"
)

var (
	addr      = flag.String("addr", "aivencloud.com:17189", "The address to bind to")
	brokers   = flag.String("brokers", os.Getenv("KAFKA_PEERS"), "The Kafka brokers to connect to, as a comma separated list")
	verbose   = flag.Bool("verbose", true, "Turn on Sarama logging")
	certFile  = flag.String("certificate", "/etc/certificates/service.cert", "The optional certificate file for client authentication")
	keyFile   = flag.String("key", "/etc/certificates/service.key", "The optional key file for client authentication")
	caFile    = flag.String("ca", "/etc/certificates/ca.pem", "The optional certificate authority file for TLS client authentication")
	verifySsl = flag.Bool("verify", true, "Optional verify ssl certificates chain")
	avroSchemaFile = flag.String("avroSchemaFile", "schemas/storage-heater-telemetry-received.avsc", "The filename of the avro schema")
	topic = flag.String("topic", "", "The kafka topic name")
)



func main() {
	flag.Parse()

	if *verbose {
		sarama.Logger = log.New(os.Stdout, "[sarama] ", log.LstdFlags)
	}

	consume()
}

func consume() {
	config := sarama.NewConfig()
	tlsConfig := createTlsConfiguration()
	if tlsConfig != nil {
		config.Net.TLS.Config = tlsConfig
		config.Net.TLS.Enable = true
	}

	consumer, err := sarama.NewConsumer([]string{ *addr }, config)

	if err != nil {
		log.Fatalln("Failed to start Sarama consumer:", err)
	}

	defer consumer.Close()

	partitionConsumer, err := consumer.ConsumePartition(*topic, 0, 0)

	defer partitionConsumer.Close()

	codec := getAvroCodec()


	for {
		msg := <- partitionConsumer.Messages()

		native, _, err := codec.NativeFromBinary(msg.Value)

		if err != nil {
			fmt.Printf("Consumed message: Key \"%s\" Value %s as offset: %d\n", BytesToString(msg.Key), native, msg.Offset)
		} else {
			fmt.Printf("Consumed message: Key \"%s\" at offset: %d but could not deserialise the data %s\n", BytesToString(msg.Key), msg.Offset, BytesToString(msg.Value))
		}
	}
}


func createTlsConfiguration() (t *tls.Config) {
	if *certFile != "" && *keyFile != "" && *caFile != "" {
		cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			log.Fatal(err)
		}

		caCert, err := ioutil.ReadFile(*caFile)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		t = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: *verifySsl,
		}
	}
	// will be nil by default if nothing is provided
	return t
}

func getAvroCodec() *goavro.Codec {

	//schemaDat, err := ioutil.ReadFile(*avroSchemaFile)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//b := make([]byte, len(schemaDat))
	//
	//var raw map[string]interface{}
	//json.Unmarshal(b, &raw)
	//jsonSchema, _ := json.Marshal(raw)
	//
	//fmt.Println(string(jsonSchema))

	codec, err:= goavro.NewCodec(`
		{
			"namespace":"Ovo.VNet.Telemetry" ,
			"name": "StorageHeaterTelemetryReceived",
			"type" : "record",
			"fields": [ 
				{
					"name": "metadata",
					"type": {
						"type": "record",
						"name": "EventMetadata",
						"fields": [
							{ "name": "eventId", "type": "string" },
							{ "name": "traceToken", "type": "string" },
							{ "name": "createdAt", "type": "long", "logicalType": "timestamp-millis" }
						]
					}
				},
				{ "name": "virtualDeviceId", "type": "string" },
				{ "name": "hardwareId", "type": "string" },
				{ "name": "assetType", "type": "int"},
				{ "name": "physicalDeviceType", "type": "int"},
				{ 
					"name": "geolocation", 
					"type" : {
						"name" : "Geolocation",
						"type" : "record", 
						"fields": [ 
							{"name": "lat", "type": "float" },
							{"name": "lon", "type": "float" } 
						]
					}
				},
				{ 
					"name": "firmware", 
					"type" : {
						"name" : "Firmware",
						"type" : "record", 
						"fields": [
						   { "name": "appVersion", "type": "string" },
						   { "name": "osVersion", "type": "string" },
						   { "name": "build", "type": "string" }, 
						]
					}
				}, 
				{
					"name": "timestamp",
					"type": "long",
					"logical-type": "timestamp-millis",
					"doc": "timestamp when telemetry was measured"
				},
				{
					"name": "powerWatts",
					"type": "long",
					"doc": "The power output in watts"
				},
				{ 
					"name": "wifiSignalStrength",
					"type": "int",
					"doc": "The wifi signal strength"
				},
				{ 
					"name": "brickTemp",
					"type": "int",
					"doc": ""
				},
				{ 
					"name": "roomTemp",
					"type": "double",
					"doc": ""
				},
				{ 
					"name": "intRoomTemp",
					"type": "double",
					"doc": ""
				}
			]
		}`)


	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	return codec
}

func BytesToString(data []byte) string {
	return string(data[:])
}



