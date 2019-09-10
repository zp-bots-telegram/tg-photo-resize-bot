workflow "Build & Push Containers" {
  on = "push"
  resolves = ["GitHub Action for Docker"]
}

action "Docker Registry" {
  uses = "actions/docker/login@fe7ed3ce992160973df86480b83a2f8ed581cd50"
  secrets = ["DOCKER_USERNAME", "DOCKER_PASSWORD"]
}

action "GitHub Action for Docker" {
  uses = "actions/docker/cli@fe7ed3ce992160973df86480b83a2f8ed581cd50"
  needs = ["Docker Registry"]
  args = "build -t zackpollard/tg-photo-resize-bot:test-$(date +%s) . && docker push zackpollard/tg-photo-resize-bot:test-$(date +%s)"
}
