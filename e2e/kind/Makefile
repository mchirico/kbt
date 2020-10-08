PROJECT = septapig
NAME = kindstarter
TAG = dev

export GOPATH=${HOME}/gopathKind
export GOBIN=${GOPATH}/bin
# Set version of kubernetes here
export K8SVERSION=v1.19.0


.PHONY: download-build
download-build:
	mkdir -p ${GOPATH}
	GO111MODULE="off" go get k8s.io/kubernetes || true
	cd ${GOPATH}/src/k8s.io/kubernetes && git checkout ${K8SVERSION}
	GO111MODULE="on" go get sigs.k8s.io/kind
#     Node image
	${GOBIN}/kind build node-image --image=${K8SVERSION}


.PHONY: create-cluster
create-cluster:
	kind delete cluster
	kind create cluster --image=${K8SVERSION} --config calico/kind-calico.yaml
	kubectl apply -f calico/ingress-nginx.yaml
	kubectl apply -f calico/tigera-operator.yaml
	kubectl apply -f calico/calicoNetwork.yaml
	kubectl apply -f calico/calicoctl.yaml
	kubectl apply -f calico/cert-manager.yaml
#	kubectl kudo init




.PHONY: packages
packages:
	kubectl kudo init
	kubectl kudo install zookeeper
	kubectl kudo install kafka
	kubectl kudo install redis
	kubectl kudo install mysql
	kubectl kudo install rabbitmq


.PHONY: sample
sample:
	rm -rf express.zip
	rm -rf node-starter-express
	curl -LO https://github.com/mchirico/node-starter/archive/express.zip
	unzip express.zip
	cd node-starter-express
	npm install


# Danger This Cleans Up Everything!
.PHONY: cleanup
cleanup:
	docker stop $(docker ps -aq)
	docker rm $(docker ps -aq)
	docker rmi $(docker images -q)


.PHONY: docker-build
docker-build:
	docker build --no-cache -t gcr.io/$(PROJECT)/$(NAME):$(TAG) -f Dockerfile .

.PHONY: kind
kind:
	kind load docker-image gcr.io/$(PROJECT)/$(NAME):$(TAG)

.PHONY: push
push:
	docker push gcr.io/$(PROJECT)/$(NAME):$(TAG) 

.PHONY: pull
pull:
	docker pull gcr.io/$(PROJECT)/$(NAME):$(TAG) 



.PHONY: run
run:
	docker run -p 3000:3000 --rm -it -d --name $(NAME) gcr.io/$(PROJECT)/$(NAME):$(TAG) 


.PHONY: runnod
runnod:
	docker run -p 3000:3000 --rm -it --name $(NAME) gcr.io/$(PROJECT)/$(NAME):$(TAG) 

.PHONY: stop
stop:
	docker stop $(NAME)


.PHONY: logs
logs:
	docker logs $(NAME)



