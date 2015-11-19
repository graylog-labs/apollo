OSs="darwin linux windows"
ARCHs="386 amd64"

for os in $OSs; do
	for arch in $ARCHs; do
		echo "Building apollo for $os-$arch"
		if [[ $os == "windows" ]]; then
			go build -o "apollo_${os}_${arch}.exe" .
		else
			go build -o "apollo_${os}_${arch}" .
		fi
	done
done
