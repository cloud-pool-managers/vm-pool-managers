package notifier

import "PoolManagerVM/backend/events"

var GlobalChan = make(chan events.RessourceEvent, 100)
