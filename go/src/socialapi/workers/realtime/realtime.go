package main

import (
	"flag"
	"koding/db/mongodb"
	"koding/db/mongodb/modelhelper"
	"koding/tools/config"
	"socialapi/db"
	realtime "socialapi/workers/realtime/lib"

	"github.com/jinzhu/gorm"
	"github.com/koding/bongo"
	"github.com/koding/broker"
	"github.com/koding/logging"
	"github.com/koding/rabbitmq"
	"github.com/streadway/amqp"
)

var (
	Bongo       *bongo.Bongo
	log         = logging.NewLogger("RealTimeWorker")
	conf        *config.Config
	flagProfile = flag.String("c", "", "Configuration profile from file")
	flagDebug   = flag.Bool("d", false, "Debug mode")
	handler     *realtime.RealtimeWorkerController
)

func main() {
	flag.Parse()
	if *flagProfile == "" {
		log.Fatal("Please define config file with -c")
	}

	conf = config.MustConfig(*flagProfile)
	setLogLevel()

	rmqConf := &rabbitmq.Config{
		Host:     conf.Mq.Host,
		Port:     conf.Mq.Port,
		Username: conf.Mq.ComponentUser,
		Password: conf.Mq.Password,
		Vhost:    conf.Mq.Vhost,
	}

	initBongo(rmqConf)
	mongo := mongodb.NewMongoDB(conf.Mongo)
	modelhelper.Initialize(conf.Mongo)
	rmq := rabbitmq.New(rmqConf, log)
	var err error
	handler, err = realtime.NewRealtimeWorkerController(rmq, mongo, log)
	if err != nil {
		panic(err)
	}

	// blocking
	realtime.Listen(rmq, startHandler)
	defer realtime.Consumer.Shutdown()
}

func startHandler() func(delivery amqp.Delivery) {
	log.Info("Worker Started to Consume")
	return func(delivery amqp.Delivery) {
		err := handler.HandleEvent(delivery.Type, delivery.Body)
		switch err {
		case nil:
			delivery.Ack(false)
		case realtime.HandlerNotFoundErr:
			log.Notice("unknown event type (%s) recieved, \n deleting message from RMQ", delivery.Type)
			delivery.Ack(false)
		case gorm.RecordNotFound:
			log.Warning("Record not found in our db (%s) recieved, \n deleting message from RMQ", string(delivery.Body))
			delivery.Ack(false)
		default:
			// add proper error handling
			// instead of puttting message back to same queue, it is better
			// to put it to another maintenance queue/exchange
			log.Error("an error occured %s, \n putting message back to queue", err)
			// multiple false
			// reque true
			delivery.Nack(false, true)
		}
	}
}

func initBongo(c *rabbitmq.Config) {
	bConf := &broker.Config{
		RMQConfig: c,
	}
	broker := broker.New(bConf, log)
	Bongo = bongo.New(broker, db.DB, log)
	Bongo.Connect()
}

func setLogLevel() {
	var logLevel logging.Level

	if *flagDebug {
		logLevel = logging.DEBUG
	} else {
		logLevel = logging.INFO
	}
	log.SetLevel(logLevel)
}
