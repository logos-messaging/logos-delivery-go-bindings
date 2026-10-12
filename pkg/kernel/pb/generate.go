package pb

//go:generate protoc -I ../../../proto --go_out=../../.. --go_opt=module=github.com/logos-messaging/logos-delivery-go-bindings logos/delivery/kernel/v1/message.proto
