# Gossip

[![Build Status](https://github.com/IdeaSeeker/GossipProtocol/workflows/CI/badge.svg)](https://github.com/IdeaSeeker/GossipProtocol/actions)

Gossip — это протокол в одноранговой компьютерной коммуникации, созданный для распространения информации по сети. Каждый узел может передавать соседям обновляемые данные, которые известны этому узлу.

### Детали реализации:

- Протокол работает поверх gRPC.
- Каждый участник протокола раз в `config.PingPeriod` отсылает доступную ему информацию о сети другому узлу. Причём выбирается тот узел, с которым дольше всего не было взаимодействия (либо вообще никогда не было). Если выбранная нода не отвечает, то она помечается удалённой.
- Публичная информация об участнике описывается в структуре [PeerInfo](info.go), приватная — в структуре [PeerData](service/service.proto).

### Запуск

- Тесты `go test -v -race ./...`
- Пример работы `go run .\example\main\main.go`

## Пример использования

```Go
const pingPeriod = 100 * time.Millisecond

func main() {
    config := gossip.PeerConfig{
        SelfAddr:   "127.0.0.1:0", // any free port
        PingPeriod: pingPeriod,
    }

    peer0, stop0, _ := gossip.SpawnNewPeer(config)
    peer1, stop1, _ := gossip.SpawnNewPeer(config)

    peer0.UpdateSelfInfo(&gossip.PeerInfo{Name: "peer0"})
    peer1.UpdateSelfInfo(&gossip.PeerInfo{Name: "peer1"})

    peer0.AddSeed(peer1.Addr())

    time.Sleep(pingPeriod * 10) // gossip sync

    log.Println("peer0 members:", peer0.GetMembers())
    log.Println("peer1 members:", peer1.GetMembers())

    stop0()

    time.Sleep(pingPeriod * 10) // gossip sync

    log.Println("peer1 members:", peer1.GetMembers())

    stop1()
}
```

После запуска кода, вывод логов в коонсоль мог бы быть таким (некоторые детали опущены для краткости):

```
127.0.0.1:53138 started
127.0.0.1:53139 started

127.0.0.1:53138 update info: &{peer0}
127.0.0.1:53139 update info: &{peer1}

127.0.0.1:53138 added a new seed 127.0.0.1:53139

127.0.0.1:53138 chose 127.0.0.1:53139 to send info
127.0.0.1:53138 sent self data
127.0.0.1:53139 added a new seed 127.0.0.1:53138
127.0.0.1:53139 received other data
127.0.0.1:53139 sent self data
127.0.0.1:53138 received other data
...
peer0 members: map[127.0.0.1:53138:{peer0} 127.0.0.1:53139:{peer1}]
peer1 members: map[127.0.0.1:53138:{peer0} 127.0.0.1:53139:{peer1}]
...
127.0.0.1:53138 cancelled context
127.0.0.1:53138 closed all connections
...
127.0.0.1:53139 deleted 127.0.0.1:53138 because of rpc error: code = Unavailable desc = error reading from server: EOF
...
peer1 members: map[127.0.0.1:53139:{peer1}]

127.0.0.1:53139 cancelled context
127.0.0.1:53139 closed all connections
```
