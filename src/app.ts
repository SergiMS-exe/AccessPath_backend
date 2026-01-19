import express, { Application } from 'express';
import cors from 'cors';
import helmet from 'helmet';
import errorHandler from './middleware/errorHandler';

import authRoutes from './routes/auth.routes';
import placeRoutes from './routes/place.routes';
import reviewRoutes from './routes/review.routes';
import featureRoutes from './routes/feature.routes';

const app: Application = express();

app.use(helmet());
app.use(cors({
  origin: process.env.ALLOWED_ORIGINS ? process.env.ALLOWED_ORIGINS.split(',') : '*',
  credentials: true
}));

app.use(express.json());
app.use(express.urlencoded({ extended: true }));

app.get('/', (req, res) => {
  res.json({
    message: 'AccessPath API',
    version: '1.0.0',
    status: 'running'
  });
});

app.use('/api/auth', authRoutes);
app.use('/api/places', placeRoutes);
app.use('/api/reviews', reviewRoutes);
app.use('/api/features', featureRoutes);

app.use(errorHandler);

export default app;
