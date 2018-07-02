package main

import (
	"fmt"
	"sync"
)

type router struct {
	ID         int
	RouteTable routeTable
	Update     chan *update
	LinkedNodes map[int]struct{}
	quit       chan int
}

type routeTable struct {
	Entries []routeTableEntry
}

type routeTableEntry struct {
	TargetId int
	NextHop  int
	//LinkId   int
	Distance int
	//TODO(xuehan): add bindWidth
}

type update struct {
	sourceRouterID int
	*routeTableEntry
}

func (r *router) sendUpdate(update *update, globalUpdateChan chan *update) error {
	globalUpdateChan <- update
	return nil
}

func (r *router) handleUpdate(update *update) error {
	targetID := update.TargetId
	nextHop := update.NextHop
	distance := update.Distance + 1

	if distance >= 16 {
		distance = 16
	}

	for _, entry := range r.RouteTable.Entries {
		if entry.TargetId == targetID {
			if entry.NextHop == nextHop{
				entry.Distance = distance
			} else {
				if entry.Distance > distance{
					entry.NextHop = nextHop
					entry.Distance = distance
				}
			}
		}
	}

	return nil
}

func (r *router) start() {
	for {
		select {
		case update := <-r.Update:
			r.handleUpdate(update)
		case <-r.quit:
			fmt.Println("router %d is closed.", r.ID)
			return
		}
	}
}

func (r *router) stop() {
	close(r.quit)
}


func main() {

	var wg sync.WaitGroup
	var quit chan int
	var globalUpdateChan = make(chan *update, 10)
	var routerMap map[int]*router

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case update := <-globalUpdateChan:
				router := routerMap[update.TargetId]
				router.Update <- update
			case <-quit:
				break
			}
		}
	}()

	fmt.Printf("start server")
}
