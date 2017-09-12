package cexio

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
)

func (a *API) ResponseCollector() {
	defer a.Close("ResponseCollector")

	a.stopDataCollector = false

	resp := &responseAction{}

	for a.stopDataCollector == false {
		a.cond.L.Lock()
		for !a.connected {
			log.Debug("DataCollector waiting...")
			a.cond.Wait()
			log.Debug("DataCollector continue...")
		}
		a.cond.L.Unlock()
		_, msg, err := a.conn.ReadMessage()
		if err != nil {
			log.Error("responseCollector, ReadMessage error: ", err.Error())
			//a.reconnect()
			a.cond.L.Lock()
			a.connected = false
			a.cond.L.Unlock()
			a.HeartBeat <- true
			a.reconnect()
			log.Debug("response: reconnect complete")
			//continue
		}

		//Send heart beat
		a.HeartBeat <- true

		err = json.Unmarshal(msg, resp)
		if err != nil {
			log.Errorf("responseCollector Unmarshal: %s\n", err, string(msg))
			log.Error("RESP Action:", resp.Action)
			//continue
		}

		subscriberIdentifier := resp.Action

		switch resp.Action {

		case "ping":
			{

				go a.pong()
				continue
			}

		case "disconnecting":
			{
				log.Info("Disconnecting...")
				log.Info("disconnecting:", string(msg))
				break
			}
		case "order-book-subscribe":
			{

				ob := &responseOrderBookSubscribe{}
				err = json.Unmarshal(msg, ob)
				if err != nil {
					log.Errorf("responseCollector | order-book-subscribe: %s\nData: %s\n", err, string(msg))
					continue
				}

				subscriberIdentifier = fmt.Sprintf("order-book-subscribe_%s", ob.Data.Pair)

				sub, err := a.subscriber(subscriberIdentifier)
				if err != nil {
					log.Error("No response handler for message: %s", string(msg))
					continue // don't know how to handle message so just skip it
				}

				sub <- ob
				continue
			}
		case "md_update":
			{

				ob := &responseOrderBookUpdate{}
				err = json.Unmarshal(msg, ob)
				if err != nil {
					log.Infof("responseCollector | md_update: %s\nData: %s\n", err, string(msg))
					continue
				}

				subscriberIdentifier = fmt.Sprintf("md_update_%s", ob.Data.Pair)

				sub, err := a.subscriber(subscriberIdentifier)
				if err != nil {
					log.Infof("No response handler for message: %s", string(msg))
					continue // don't know how to handle message so just skip it
				}

				sub <- ob
				continue
			}
		case "get-balance":
			{
				ob := &responseGetBalance{}
				err = json.Unmarshal(msg, ob)
				if err != nil {
					log.Infof("responseCollector | get_balance: %s\nData: %s\n", err, string(msg))
					continue
				}

				subscriberIdentifier = "get-balance"

				sub, err := a.subscriber(subscriberIdentifier)
				if err != nil {
					log.Infof("No response handler for message: %s", string(msg))
					continue // don't know how to handle message so just skip it
				}

				sub <- ob
				continue
			}

		default:
			sub, err := a.subscriber(subscriberIdentifier)
			if err != nil {
				log.Errorf("No response handler for message: %s", string(msg))
				continue // don't know how to handle message so just skip it
			}
			//log.Debug("Sending response:", string(msg))
			a.HeartBeat <- true
			sub <- msg

		}
	}

}

func (a *API) connectionResponse() {

	resp := &responseAction{}

	for !a.connected {

		_, msg, err := a.conn.ReadMessage()
		if err != nil {
			log.Error("Error while waiting for conection start: ", err.Error())
			return
		}
		err = json.Unmarshal(msg, resp)
		if err != nil {
			log.Fatal("connection start error response: %s\n  Data: %s\n", err, string(msg))
		}

		subscriberIdentifier := resp.Action

		switch resp.Action {

		case "ping":
			{

				a.pong()
				continue
			}

		case "disconnecting":
			{
				log.Info("Disconnecting...")
				log.Info("disconnecting:", string(msg))
				break
			}
		case "connected":
			{
				log.Debug("Conection message detected...")
				sub, err := a.subscriber(subscriberIdentifier)
				if err != nil {
					log.Infof("No response handler for message: %s", string(msg))
					continue // don't know how to handle message so just skip it
				}
				log.Debug("Connection response: ", string(msg))
				sub <- msg
			}

		case "auth":
			log.Debug("Auth message detected...")
			sub, err := a.subscriber(subscriberIdentifier)
			if err != nil {
				log.Infof("No response handler for message: %s", string(msg))
				continue // don't know how to handle message so just skip it
			}
			log.Debug("Connection response: ", string(msg))
			a.connected = true
			sub <- msg
			break

		default:
			{
				log.Fatal("unexpected message recieved: ", string(msg))
			}
		}
	}

}