/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	// _ "k8s.io/code-generator/cmd/codegen"

	"github.com/go-redis/redis"
	"github.com/golang/glog"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/ory/ladon"
	sqlmanager "github.com/ory/ladon/manager/sql"
	redismanager "github.com/wehco/ladon-community/manager/redis"

	clientset "github.com/kminehart/ladon-resource-manager/pkg/client/clientset/versioned"
	informers "github.com/kminehart/ladon-resource-manager/pkg/client/informers/externalversions"
	"github.com/kminehart/ladon-resource-manager/pkg/signals"
)

var (
	driver      string
	postgresURL string
	redisURL    string
	redisPrefix string
	masterURL   string
	kubeconfig  string
)

func main() {
	flag.Parse()

	var m ladon.Manager

	// Because some users might include both redis and postgres params, we'll prefer the more well-supported Postgres manager
	if postgresURL == "" {
		postgresURL = os.Getenv("POSTGRES_URL")
	}
	if redisURL == "" {
		redisURL = os.Getenv("REDIS_URL")
	}

	if postgresURL != "" {
		// Connect to the SQL server to create policies...
		ladonDB, err := sqlx.Open("postgres", postgresURL)
		if err != nil {
			glog.Fatal(err.Error())
		}
		defer func() {
			if err := ladonDB.Close(); err != nil {
				glog.Fatalln("Error closing ladon database. Error:", err.Error())
			}
		}()
		sqlm := sqlmanager.NewSQLManager(ladonDB, nil)
		if _, err := sqlm.CreateSchemas("ladon", "ladon_migrations"); err != nil {
			glog.Fatal(err.Error())
		}
		m = sqlm
	} else if redisURL != "" {
		redisOptions, err := redis.ParseURL(redisURL)
		if err != nil {
			glog.Fatalln("Error parsing redis URL. Error:", err.Error())
		}

		redisClient := redis.NewClient(redisOptions)
		m = redismanager.NewRedisManager(redisClient, redisPrefix)
	}

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	ladonClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building ladon manager clientset: %s", err.Error())
	}

	var (
		kubeInformerFactory  = kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
		ladonInformerFactory = informers.NewSharedInformerFactory(ladonClient, time.Second*30)
		controller           = NewController(kubeClient, ladonClient, kubeInformerFactory, ladonInformerFactory, m)
	)

	go kubeInformerFactory.Start(stopCh)
	go ladonInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&postgresURL, "postgres-url", "", "The URL of the postgres server.")
	flag.StringVar(&redisURL, "redis-url", "", "The connection string for the redis server")
	flag.StringVar(&redisPrefix, "redis-prefix", "ladon", "The key prefix used for ladon keys")
}
