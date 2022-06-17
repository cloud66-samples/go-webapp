#!/usr/bin/ruby

BACKENDS = %w[web1 web2]

# get the backend status. returns true if the backend has an IP (indicating it's running)
def get_backend_status(backend)
  puts "Getting status for #{backend}"

  status = `echo "show servers conn web" | socat stdio tcp4-connect:127.0.0.1:9999 | awk '{split($0,a," "); print a[1],a[3]}' | grep "web/#{backend}" | awk '{split($0,a," "); print a[2]}'`
  running = status.chomp != '-'

  puts "Status for #{backend} is #{running ? 'running' : 'stopped'}"
  running
end

def start_backend(backend)
  puts "Starting #{backend}"

  `docker rm #{backend}`
  `docker run -d --name #{backend} --net sample go-webapp /go/src/go-webapp/go-webapp`
end

def stop_backend(backend)
  puts "Stopping #{backend}"

  `docker stop #{backend}`
end

def swap_backends
  # get the backend that's not running
  backend = BACKENDS.find { |b| !get_backend_status(b) }
  # start it
  start_backend(backend)
  # stop the other backend
  stop_backend(BACKENDS.find { |b| b != backend })
end

def start_haproxy
  # check if haproxy container is running
  running = `docker ps | grep haproxy | wc -l`.chomp.to_i > 0
  puts "Haproxy is #{running ? 'running' : 'stopped'}"

  unless running
    puts 'Starting haproxy'

    `docker rm haproxy`
    `docker run -d --name haproxy --net sample -v $(pwd):/usr/local/etc/haproxy:ro -p 4500:4500 -p 9999:9999 haproxytech/haproxy-alpine`
  end
end

def stop_all
  puts 'Stopping haproxy and backends'

  `docker stop haproxy`
  BACKENDS.each { |b| stop_backend(b) }
end

# this tool takes in 1 arg as the command: start, stop or deploy
# start: starts the haproxy
# stop: stops the haproxy
# deploy: swaps the current backend with a new one or starts a new one if none exists
if ARGV[0] == 'start'
  start_haproxy
elsif ARGV[0] == 'deploy'
  swap_backends
elsif ARGV[0] == 'stop'
  stop_all
else
  puts 'Usage: deploy.rb start|stop|deploy'
end
