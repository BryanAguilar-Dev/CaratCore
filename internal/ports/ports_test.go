package ports

import (
	"errors"
	"reflect"
	"testing"
)

func fakeReadFile(files map[string]string) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		data, ok := files[path]
		if !ok {
			return nil, errors.New("not found")
		}
		return []byte(data), nil
	}
}

func TestInUse(t *testing.T) {
	files := map[string]string{
		"/proc/net/tcp": "  sl  local_address rem_address   st tx_queue rx_queue\n" +
			"   0: 00000000:0BB8 00000000:0000 0A 00000000:00000000 00:00000000\n" + // 3000 LISTEN
			"   1: 0100007F:0BB9 00000000:0000 01 00000000:00000000 00:00000000\n", // 3001 ESTABLISHED
		"/proc/net/udp": "  sl  local_address rem_address   st\n" +
			"   0: 0100007F:0035 00000000:0000 07 00000000:00000000\n", // 53 bound
	}
	got, err := InUse(fakeReadFile(files))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"3000/tcp": true, "53/udp": true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestInUseWithoutProc(t *testing.T) {
	if _, err := InUse(fakeReadFile(nil)); err == nil {
		t.Fatal("want an error when /proc/net cannot be read")
	}
}
