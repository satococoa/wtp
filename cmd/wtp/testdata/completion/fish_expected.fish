# wtp fish shell completion

function __fish_wtp_dynamic_complete --description 'wtp dynamic completion helper'
	set -l tokens (commandline -opc)
	set -l args
	set -l token_count (count $tokens)
	if test $token_count -gt 1
		set args $tokens[2..-1]
	end

	set -l current (commandline -ct)

	if test -n "$current"
		if string match -q -- '-*' $current
			set args $args $current
		end
	end

	set args $args --generate-shell-completion

	if not command -sq wtp
		return
	end

	set -l raw (command wtp $args)
	for line in $raw
		if test -z "$line"
			continue
		end

		set -l parts (string split -m 1 ":" -- $line)
		if test (count $parts) -gt 1
			set -l remainder $parts[2]
			if string match -q "* *" $remainder
				echo $parts[1]
				continue
			end
		end

		echo $line
	end
end

complete -c wtp -f -a '(__fish_wtp_dynamic_complete)'
