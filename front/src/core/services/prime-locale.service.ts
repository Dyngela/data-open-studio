import { Injectable } from '@angular/core';
import { PrimeNG } from 'primeng/config';

@Injectable({
  providedIn: 'root'
})
export class PrimeLocaleService {

  setLocale(config: PrimeNG, locale: string): void {
    switch (locale.toLowerCase()) {
      case 'fr': {
        config.setTranslation({
          startsWith: 'Commence par',
          contains: 'Contient',
          notContains: 'Ne contient pas',
          endsWith: 'Se termine par',
          equals: 'Égal',
          notEquals: 'Différent',
          noFilter: 'Aucun filtre',
          lt: 'Inférieur à',
          lte: 'Inférieur ou égal à',
          gt: 'Supérieur à',
          gte: 'Supérieur ou égal à',
          is: 'Est',
          isNot: "N'est pas",
          before: 'Avant',
          after: 'Après',
          apply: 'Appliquer',
          matchAll: 'Correspond à tous',
          matchAny: "Correspond à n'importe lequel",
          addRule: 'Ajouter une règle',
          removeRule: 'Supprimer la règle',
          accept: 'Oui',
          reject: 'Non',
          choose: 'Choisir',
          upload: 'Télécharger',
          cancel: 'Annuler',
          dayNames: ['Dimanche', 'Lundi', 'Mardi', 'Mercredi', 'Jeudi', 'Vendredi', 'Samedi'],
          dayNamesShort: ['Dim', 'Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam'],
          dayNamesMin: ['Di', 'Lu', 'Ma', 'Me', 'Je', 'Ve', 'Sa'],
          monthNames: [
            'Janvier', 'Février', 'Mars', 'Avril', 'Mai', 'Juin',
            'Juillet', 'Août', 'Septembre', 'Octobre', 'Novembre', 'Décembre'
          ],
          monthNamesShort: [
            'Jan', 'Fév', 'Mar', 'Avr', 'Mai', 'Juin',
            'Juil', 'Août', 'Sep', 'Oct', 'Nov', 'Déc'
          ],
          today: "Aujourd'hui",
          clear: 'Effacer',
          weekHeader: 'Sem',
          firstDayOfWeek: 1,
          dateFormat: 'dd/mm/yy',
          weak: 'Faible',
          medium: 'Moyen',
          strong: 'Fort',
          passwordPrompt: 'Entrez un mot de passe',
          emptyMessage: 'Aucun résultat trouvé',
          emptyFilterMessage: 'Aucun résultat trouvé'
        });
        break;
      }
      case 'en': {
        // PrimeNG est déjà en anglais par défaut
        break;
      }
      default: {
        console.warn(`Locale '${locale}' non disponible, utilisation de la locale par défaut.`);
        break;
      }
    }
  }
}
