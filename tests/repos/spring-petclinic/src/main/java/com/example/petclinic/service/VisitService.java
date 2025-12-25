package com.example.petclinic.service;

import com.example.petclinic.dto.VisitDto;
import com.example.petclinic.exception.ResourceNotFoundException;
import com.example.petclinic.model.Pet;
import com.example.petclinic.model.Vet;
import com.example.petclinic.model.Visit;
import com.example.petclinic.repository.PetRepository;
import com.example.petclinic.repository.VetRepository;
import com.example.petclinic.repository.VisitRepository;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import java.time.LocalDate;
import java.util.List;
import java.util.stream.Collectors;

/**
 * Service layer for Visit operations.
 */
@Service
@Transactional
public class VisitService {

    private final VisitRepository visitRepository;
    private final PetRepository petRepository;
    private final VetRepository vetRepository;

    public VisitService(VisitRepository visitRepository,
                        PetRepository petRepository,
                        VetRepository vetRepository) {
        this.visitRepository = visitRepository;
        this.petRepository = petRepository;
        this.vetRepository = vetRepository;
    }

    /**
     * Find all visits.
     */
    @Transactional(readOnly = true)
    public List<VisitDto> findAll() {
        return visitRepository.findAll().stream()
            .map(this::toDto)
            .collect(Collectors.toList());
    }

    /**
     * Find visit by ID.
     */
    @Transactional(readOnly = true)
    public VisitDto findById(Long id) {
        Visit visit = visitRepository.findById(id)
            .orElseThrow(() -> new ResourceNotFoundException("Visit", "id", id));
        return toDto(visit);
    }

    /**
     * Find visits for a pet.
     */
    @Transactional(readOnly = true)
    public List<VisitDto> findByPetId(Long petId) {
        return visitRepository.findByPetId(petId).stream()
            .map(this::toDto)
            .collect(Collectors.toList());
    }

    /**
     * Find visits for a vet.
     */
    @Transactional(readOnly = true)
    public List<VisitDto> findByVetId(Long vetId) {
        return visitRepository.findByVetId(vetId).stream()
            .map(this::toDto)
            .collect(Collectors.toList());
    }

    /**
     * Find today's visits.
     */
    @Transactional(readOnly = true)
    public List<VisitDto> findTodaysVisits() {
        return visitRepository.findTodaysVisits(LocalDate.now()).stream()
            .map(this::toDto)
            .collect(Collectors.toList());
    }

    /**
     * Schedule a new visit.
     */
    public VisitDto scheduleVisit(VisitDto visitDto) {
        Pet pet = petRepository.findById(visitDto.petId())
            .orElseThrow(() -> new ResourceNotFoundException("Pet", "id", visitDto.petId()));

        Visit visit = new Visit();
        visit.setDate(visitDto.date());
        visit.setDescription(visitDto.description());
        visit.setPet(pet);

        if (visitDto.vetId() != null) {
            Vet vet = vetRepository.findById(visitDto.vetId())
                .orElseThrow(() -> new ResourceNotFoundException("Vet", "id", visitDto.vetId()));
            visit.setVet(vet);
        }

        Visit savedVisit = visitRepository.save(visit);
        return toDto(savedVisit);
    }

    /**
     * Update visit description.
     */
    public VisitDto updateDescription(Long id, String description) {
        Visit visit = visitRepository.findById(id)
            .orElseThrow(() -> new ResourceNotFoundException("Visit", "id", id));

        visit.setDescription(description);
        Visit savedVisit = visitRepository.save(visit);
        return toDto(savedVisit);
    }

    /**
     * Cancel (delete) a visit.
     */
    public void cancelVisit(Long id) {
        if (!visitRepository.existsById(id)) {
            throw new ResourceNotFoundException("Visit", "id", id);
        }
        visitRepository.deleteById(id);
    }

    /**
     * Find visits in date range.
     */
    @Transactional(readOnly = true)
    public List<VisitDto> findByDateRange(LocalDate startDate, LocalDate endDate) {
        return visitRepository.findByDateBetween(startDate, endDate).stream()
            .map(this::toDto)
            .collect(Collectors.toList());
    }

    private VisitDto toDto(Visit visit) {
        return new VisitDto(
            visit.getId(),
            visit.getDate(),
            visit.getDescription(),
            visit.getPet() != null ? visit.getPet().getId() : null,
            visit.getPet() != null ? visit.getPet().getName() : null,
            visit.getVet() != null ? visit.getVet().getId() : null,
            visit.getVet() != null ? visit.getVet().getFullName() : null
        );
    }
}
