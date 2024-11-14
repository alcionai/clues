package mock

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strconv"

	"github.com/alcionai/clues"
	"github.com/alcionai/clues/clog"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

var stubValues = map[int]any{}

type stub struct {
	m1 any
	m2 any
	m3 any
	m4 any
	m5 any
}

func stubStruct(i int) *stub {
	return &stub{
		m1: stubValues[i-1],
		m2: stubValues[i-2],
		m3: stubValues[i-3],
		m4: stubValues[i-4],
		m5: stubValues[i-5],
	}
}

var singleton *server

type server struct {
	test *httptest.Server
	idxs []int
}

func StartService(ctx context.Context) *server {
	if singleton != nil {
		return singleton
	}

	for i := range 1024 {
		switch 0 {
		case i%3 + i%5:
			stubValues[i] = stubStruct(i)
		case i % 3:
			stubValues[i] = strconv.Itoa(i)
		case i % 5:
			stubValues[i] = i%2 == 0
		default:
			stubValues[i] = i
		}
	}

	router := httprouter.New()
	router.GET("/hello/:name", do)

	srv := httptest.NewServer(router)

	stubSrv := server{
		test: srv,
		idxs: []int{},
	}

	for range 20 {
		r := rand.IntN(950) + 5
		stubSrv.idxs = append(stubSrv.idxs, r)
	}

	singleton = &stubSrv

	return &stubSrv
}

func (s server) Call(
	name string,
) error {
	req, err := http.NewRequest("/"+name, s.test.URL, nil)
	if err != nil {
		return errors.Wrap(err, "req creation")
	}

	http.DefaultClient.Do(req)

	return nil
}

func do(
	w http.ResponseWriter,
	req *http.Request,
	params httprouter.Params,
) {
	ctx := req.Context()
	ctx = clues.Add(ctx, "name", params.ByName("name"))

	willErr(ctx, 0)
	willErr(ctx, 5)
	willOK(ctx, 10)
	willOK(ctx, 15)
}

func willErr(ctx context.Context, i int) {
	err := willErr2(ctx, i)
	clog.CtxErr(ctx, err).Error("err:", clues.ToCore(err))
}

func willErr2(ctx context.Context, i int) error {
	ctx = clues.Add(ctx, i, stubValues[i])
	return clues.Stack(willErr3(ctx, i+1)).OrNil()
}

func willErr3(ctx context.Context, i int) error {
	ctx = clues.Add(ctx, i, stubValues[i])
	return clues.Stack(willErr4(ctx, i+1)).OrNil()
}

func willErr4(ctx context.Context, i int) error {
	ctx = clues.Add(ctx, i, stubValues[i])
	return clues.Stack(willErr5(ctx, i+1)).OrNil()
}

func willErr5(ctx context.Context, i int) error {
	return clues.NewWC(ctx, "forced").With(
		i, stubValues[i],
		i+1, stubValues[i+1])
}

func willOK(ctx context.Context, i int) {
	// fan out
	for iter := range 10 {
		ictx := clues.Add(ctx, "iter", iter)

		if err := willOK2(ictx, i); err != nil {
			fmt.Println("error", clues.ToCore(err))
		}
	}
}

func willOK2(ctx context.Context, i int) error {
	ctx = clues.Add(ctx, i, stubValues[i])
	return willOK3(ctx, i+1)
}

func willOK3(ctx context.Context, i int) error {
	ctx = clues.Add(ctx, i, stubValues[i])
	return willOK4(ctx, i+1)
}

func willOK4(ctx context.Context, i int) error {
	ctx = clues.Add(ctx, i, stubValues[i])
	return willOK5(ctx, i+1)
}

func willOK5(ctx context.Context, i int) error {
	clues.Add(
		ctx,
		i, stubValues[i],
		i+1, stubValues[i+1])

	clog.Ctx(ctx).Error("yo")

	fmt.Println(clues.NewWC(ctx, "dropped"))

	return nil
}
