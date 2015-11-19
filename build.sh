OSs="darwin linux windows"
ARCHs="386 amd64"

for os in $OSs; do
	for arch in $ARCHs; do
		go build -o apollo-collector-$os-$arch .
	done
done
