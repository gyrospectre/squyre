## Testing
sam build
sam local invoke GreynoiseFunction --event events/address.json 
sam local invoke IPAPIFunction --event events/address.json 
