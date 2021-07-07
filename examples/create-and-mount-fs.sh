#!/bin/sh

timestamp="$(date +%Y-%m-%d)T00:00"

datasets="home
root/VOID
root/ARCH
root/DEBIAN
lxc/test-1"

mkcommand() {
	file="$1"
	command="$2"
	size_cmd="$3"
	dirname="#${file}#"
	mkdir -p "${dirname}"

	printf '#!/bin/sh\n\n\n%s\n' "$command" > "${dirname}/cmd"
	printf '#!/bin/sh\n\n\n%s\n' "$size_cmd" > "${dirname}/size"
	chmod +x "${dirname}/cmd"
	chmod +x "${dirname}/size"
}

mkdir -p data-origin
cd data-origin

for dataset in $datasets; do
        outfilename="${dataset}"
        while [ "${outfilename#*/}" != "$outfilename" ]; do
                outfilename="${outfilename%%/*}-${outfilename#*/}";
        done
        dataset="tank/${dataset}@${timestamp}"
	# In tests, wc -c was faster than dd of=/dev/null
        mkcommand "${outfilename}@${timestamp}" "zfs send '${dataset}'" "zfs send '${dataset}' | wc -c"
done

cd ..
mkdir -p data
shellfs data data-origin &
