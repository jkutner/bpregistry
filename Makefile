export PORT=5001

default:
	echo "Please run a specific target as this first one will do nothing."

checks:
ifndef REGISTRY
	echo "REGISTRY not set"
	exit 1
endif
ifndef ECR_REGISTRY
	echo "ECR_REGISTRY not set"
	exit 1
endif

run: checks
	REGISTRY=$(REGISTRY) ECR_REGISTRY=$(ECR_REGISTRY) PORT=$(PORT) go run .