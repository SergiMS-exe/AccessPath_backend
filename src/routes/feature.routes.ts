import { Router } from 'express';
import FeatureController from '../controllers/feature.controller';

const router = Router();

router.get('/', FeatureController.getAll);

router.get('/:id', FeatureController.getById);

router.get('/category/:category', FeatureController.getByCategory);

export default router;
