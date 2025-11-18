import { PoolServiceClient } from '../grpc/FrontcontrolServiceClientPb';
import { GetPoolRequest } from '../grpc/frontcontrol_pb';

// URL du proxy Caddy
const client = new PoolServiceClient("https://localhost:443", null, null);

const req = new GetPoolRequest();
req.setUser("testuser");
req.setPoolId("mypool");

client.getPool(req, {}, (err, response) => {
  if (err) {
    console.error("❌ Erreur gRPC :", err.message || err);
    return;
  }

  console.log("✅ Réponse :", response.toObject());
});