cd /var/www/app/
go build -o go-hello-server main.go

sudo mv go-hello-server /usr/local/bin/

Reload the systemd manager configuration:
sudo systemctl daemon-reload

Start the service:
sudo systemctl start go-hello-server

Enable the service to start on boot:
sudo systemctl enable go-hello-server

You can check the status of your service with:
sudo systemctl status go-hello-server

And view the logs with:
sudo journalctl -u go-hello-server